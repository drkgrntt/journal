// Pure logic for the offline outbox (see outbox.js for storage and network).
// Plain data in, plain data out - no DOM, localStorage or fetch - so it can be
// tested in Node. Every function returns a new record and never mutates its
// input.
//
// record = { ops: Op[], idMap: { [clientId]: serverId } }
// Op = { id, kind: "action"|"thankful", type: "create"|"complete"|"uncomplete"|"delete",
//        ref: { clientId?, serverId? }, text?, completedAt? (ms), isComplete?, createdAt, rejected?, sent? }
// `sent` marks a create whose request has gone out (outbox.js persists it
// before the fetch). If the response is lost the server may still hold the row,
// so such a create must stay queued: replaying it is deduplicated by clientId.

export function emptyRecord() {
	return { ops: [], idMap: {} }
}

const BASE = { action: "/api/action-items", thankful: "/api/thankfuls" }

function resolveServerId(ref, idMap) {
	return ref.serverId || (ref.clientId && idMap[ref.clientId]) || ""
}

// Which item an op is about. A synced item is keyed by its server id (so a
// clientId that has since synced and a raw serverId match); one whose create
// is still queued is keyed by its clientId.
function itemKey(op, idMap) {
	const serverId = resolveServerId(op.ref, idMap)
	return `${op.kind}:${serverId ? `s:${serverId}` : `c:${op.ref.clientId}`}`
}

// Drops idMap entries nothing in the queue refers to any more, so an empty
// queue leaves an empty record.
function prune(record) {
	const used = new Set(record.ops.map((o) => o.ref.clientId).filter(Boolean))
	const idMap = {}
	for (const [k, v] of Object.entries(record.idMap)) if (used.has(k)) idMap[k] = v
	return { ops: record.ops, idMap }
}

// Flags a create as sent. Call (and persist) before the request goes out.
export function markSent(record, opId) {
	return { ops: record.ops.map((o) => (o.id === opId ? { ...o, sent: true } : o)), idMap: record.idMap }
}

// Adds an op, folding it into what is already queued so a flaky connection
// doesn't replay a pile of redundant requests. Order is otherwise preserved.
export function enqueue(record, op) {
	return prune(enqueueRaw(record, op))
}

function enqueueRaw(record, op) {
	const idMap = record.idMap
	let ops = record.ops.slice()

	if (op.type === "create") {
		return { ops: [...ops, op], idMap }
	}

	const serverId = resolveServerId(op.ref, idMap)
	const target = { ...op, ref: serverId ? { serverId } : { clientId: op.ref.clientId } }
	const key = itemKey(target, idMap)
	const same = (o) => itemKey(o, idMap) === key

	const i = serverId ? -1 : ops.findIndex((o) => o.type === "create" && same(o))
	if (!serverId && i === -1) return { ops, idMap } // nothing to act on

	// A create never sent can be edited or cancelled in place. One that was
	// sent stays queued (the server may have it); ops go after it instead.
	if (i !== -1 && !ops[i].sent) {
		if (op.type === "delete") return { ops: ops.filter((o) => !same(o)), idMap }
		const create = { ...ops[i] }
		if (op.type === "complete") {
			create.isComplete = true
			create.completedAt = op.completedAt
		} else {
			create.isComplete = false
			delete create.completedAt
		}
		ops[i] = create
		return { ops, idMap }
	}

	// Already deleting it: later ops for the item are moot.
	if (ops.some((o) => o.type === "delete" && same(o))) return { ops, idMap }

	// Never drop the create itself here (see above).
	if (op.type === "delete") {
		ops = ops.filter((o) => o.type === "create" || !same(o))
	} else {
		ops = ops.filter((o) => !(same(o) && (o.type === "complete" || o.type === "uncomplete")))
	}
	return { ops: [...ops, target], idMap }
}

// The HTTP request for an op, or null when it targets an item whose server id
// isn't known yet (the caller keeps it for a later flush).
export function buildRequest(op, idMap) {
	const base = BASE[op.kind]
	const params = new URLSearchParams()

	if (op.type === "create") {
		params.set("text", op.text ?? "")
		params.set("clientId", op.ref.clientId)
		if (op.isComplete) {
			params.set("isComplete", "true")
			if (op.completedAt != null) params.set("completedAt", String(op.completedAt))
		}
		return { method: "POST", url: base, body: params.toString() }
	}

	const serverId = resolveServerId(op.ref, idMap)
	if (!serverId) return null
	const url = `${base}/${encodeURIComponent(serverId)}`

	if (op.type === "delete") return { method: "DELETE", url, body: "" }

	// The server requires text on every PUT.
	params.set("text", op.text ?? "")
	if (op.type === "complete") {
		params.set("isComplete", "true")
		if (op.completedAt != null) params.set("completedAt", String(op.completedAt))
	} else {
		params.set("isComplete", "false")
	}
	return { method: "PUT", url, body: params.toString() }
}

// Maps an HTTP response to a send status. A create answered without the row's
// id is treated as lost: keeping it queued is safe because the replay hits the
// server's clientId dedup and returns the id, whereas removing it would leave
// no idMap entry for later ops.
export function classifyResponse(op, res, itemId) {
	if (res.type === "opaqueredirect" || res.status === 401) return { status: "auth" }
	if (res.status >= 500 || res.status === 408 || res.status === 429) return { status: "retry" }
	if (res.status === 404 && op.type !== "create") return { status: "gone" }
	if (!res.ok) return { status: "rejected" }
	if (op.type === "create" && !itemId) return { status: "retry" }
	return { status: "saved", serverId: itemId || undefined }
}

// Folds one send's outcome back into the record. `op` is what was sent;
// the record may have changed since (enqueue runs during a flush).
export function applyResult(record, op, result) {
	const { status } = result
	const idMap = record.idMap

	if (status === "rejected") {
		return { ops: record.ops.map((o) => (o.id === op.id ? { ...o, rejected: true } : o)), idMap }
	}
	if (status === "retry" || status === "auth") return record

	// saved or gone
	const ops = record.ops.filter((o) => o.id !== op.id)
	if (status === "saved" && op.type === "create" && result.serverId) {
		return prune({ ops, idMap: { ...idMap, [op.ref.clientId]: result.serverId } })
	}
	return prune({ ops, idMap })
}

// Removes rejected ops older than maxAgeMs - they would otherwise hold
// plaintext on the device forever - along with anything queued against a
// dropped create.
export function dropRejected(record, maxAgeMs, now) {
	const dropped = record.ops.filter((o) => o.rejected && now - o.createdAt > maxAgeMs)
	if (dropped.length === 0) return record
	const gone = new Set(dropped.map((o) => o.id))
	const deadClients = new Set(dropped.filter((o) => o.type === "create").map((o) => o.ref.clientId))
	const ops = record.ops.filter((o) => !gone.has(o.id) && !(o.type !== "create" && !o.ref.serverId && deadClients.has(o.ref.clientId)))
	return prune({ ops, idMap: record.idMap })
}

export function pendingCount(record) {
	return record.ops.length
}

// Ops to send now, in order. A rejected op is skipped once it has been tried
// this page load, so a bad one can't loop.
export function nextOps(record, attempted) {
	return record.ops.filter((o) => !(o.rejected && attempted.has(o.id)))
}
