// Run: node --test internal/web/assets/js/utils/outboxQueue.test.js
import { test } from "node:test"
import assert from "node:assert/strict"
import { emptyRecord, enqueue, buildRequest, applyResult, pendingCount, nextOps, markSent, classifyResponse, dropRejected } from "./outboxQueue.js"

let n = 0
const op = (o) => ({ id: `op${++n}`, createdAt: n, ...o })
const create = (clientId, text = "t", kind = "action") => op({ kind, type: "create", ref: { clientId }, text })
const act = (type, ref, extra = {}) => op({ kind: "action", type, ref, text: "t", ...extra })
const all = (ops, rec = emptyRecord()) => ops.reduce((r, o) => enqueue(r, o), rec)
const types = (rec) => rec.ops.map((o) => o.type)

test("enqueue keeps replay order", () => {
	const rec = all([create("a"), create("b", "x", "thankful"), act("complete", { serverId: "s1" }, { completedAt: 1 })])
	assert.deepEqual(types(rec), ["create", "create", "complete"])
	assert.deepEqual(rec.ops.map((o) => o.ref.clientId ?? o.ref.serverId), ["a", "b", "s1"])
})

test("complete folds into an unsent create", () => {
	const rec = all([create("a"), act("complete", { clientId: "a" }, { completedAt: 123 })])
	assert.equal(rec.ops.length, 1)
	assert.equal(rec.ops[0].type, "create")
	assert.equal(rec.ops[0].isComplete, true)
	assert.equal(rec.ops[0].completedAt, 123)
})

test("uncomplete clears completion on an unsent create", () => {
	const rec = all([create("a"), act("complete", { clientId: "a" }, { completedAt: 123 }), act("uncomplete", { clientId: "a" })])
	assert.equal(rec.ops.length, 1)
	assert.equal(rec.ops[0].isComplete, false)
	assert.equal(rec.ops[0].completedAt, undefined)
})

test("delete removes an unsent create and sends nothing", () => {
	const rec = all([create("a"), create("b"), act("delete", { clientId: "a" })])
	assert.deepEqual(rec.ops.map((o) => o.ref.clientId), ["b"])
})

test("complete replaces earlier pending complete/uncomplete for a synced item", () => {
	const rec = all([
		act("complete", { serverId: "s1" }, { completedAt: 1 }),
		act("uncomplete", { serverId: "s1" }),
		act("complete", { serverId: "s1" }, { completedAt: 3 }),
	])
	assert.equal(rec.ops.length, 1)
	assert.equal(rec.ops[0].completedAt, 3)
})

test("replacement only touches the same item", () => {
	const rec = all([act("complete", { serverId: "s1" }, { completedAt: 1 }), act("complete", { serverId: "s2" }, { completedAt: 2 })])
	assert.equal(rec.ops.length, 2)
})

test("delete on a synced item drops earlier ops for it and queues one delete", () => {
	const rec = all([
		act("complete", { serverId: "s1" }, { completedAt: 1 }),
		act("complete", { serverId: "s2" }, { completedAt: 1 }),
		act("delete", { serverId: "s1" }),
	])
	assert.deepEqual(types(rec), ["complete", "delete"])
	assert.deepEqual(rec.ops.map((o) => o.ref.serverId), ["s2", "s1"])
})

test("ops for an item with a pending delete are dropped", () => {
	const rec = all([act("delete", { serverId: "s1" }), act("complete", { serverId: "s1" }, { completedAt: 1 }), act("delete", { serverId: "s1" })])
	assert.deepEqual(types(rec), ["delete"])
})

test("action and thankful with the same serverId are different items", () => {
	const rec = all([act("delete", { serverId: "s1" }), op({ kind: "thankful", type: "delete", ref: { serverId: "s1" } })])
	assert.equal(rec.ops.length, 2)
})

test("clientId resolves through idMap to a serverId", () => {
	const base = { ops: [], idMap: { a: "s1" } }
	// the map entry is pruned once no op needs it, so a later op must use the serverId
	let rec = enqueue(base, act("complete", { clientId: "a" }, { completedAt: 5 }))
	assert.equal(rec.ops[0].ref.serverId, "s1")
	assert.deepEqual(rec.idMap, {})
	rec = enqueue(rec, act("uncomplete", { serverId: "s1" }))
	assert.equal(rec.ops.length, 1)
	assert.equal(rec.ops[0].type, "uncomplete")
})

test("op for an unknown clientId is dropped", () => {
	assert.equal(all([act("complete", { clientId: "zzz" }, { completedAt: 1 })]).ops.length, 0)
})

test("enqueue does not mutate its input", () => {
	const rec = all([create("a")])
	const snapshot = structuredClone(rec)
	enqueue(rec, act("complete", { clientId: "a" }, { completedAt: 1 }))
	enqueue(rec, act("delete", { clientId: "a" }))
	assert.deepEqual(rec, snapshot)
})

test("buildRequest: every kind and type", () => {
	const form = (b) => Object.fromEntries(new URLSearchParams(b))
	let r = buildRequest(create("c1", "hi"), {})
	assert.deepEqual([r.method, r.url, form(r.body)], ["POST", "/api/action-items", { text: "hi", clientId: "c1" }])

	r = buildRequest({ ...create("c1", "hi"), isComplete: true, completedAt: 99 }, {})
	assert.deepEqual(form(r.body), { text: "hi", clientId: "c1", isComplete: "true", completedAt: "99" })

	r = buildRequest(create("c2", "thx", "thankful"), {})
	assert.deepEqual([r.method, r.url, form(r.body)], ["POST", "/api/thankfuls", { text: "thx", clientId: "c2" }])

	r = buildRequest(act("complete", { serverId: "s1" }, { text: "hi", completedAt: 7 }), {})
	assert.deepEqual([r.method, r.url, form(r.body)], ["PUT", "/api/action-items/s1", { text: "hi", isComplete: "true", completedAt: "7" }])

	r = buildRequest(act("uncomplete", { serverId: "s1" }, { text: "hi" }), {})
	assert.deepEqual([r.method, r.url, form(r.body)], ["PUT", "/api/action-items/s1", { text: "hi", isComplete: "false" }])

	r = buildRequest(act("delete", { serverId: "s1" }), {})
	assert.deepEqual([r.method, r.url, r.body], ["DELETE", "/api/action-items/s1", ""])

	r = buildRequest(op({ kind: "thankful", type: "delete", ref: { serverId: "s9" } }), {})
	assert.deepEqual([r.method, r.url], ["DELETE", "/api/thankfuls/s9"])
})

test("buildRequest resolves clientId via idMap, null when unresolved", () => {
	const o = act("delete", { clientId: "a" })
	assert.equal(buildRequest(o, {}), null)
	assert.equal(buildRequest(o, { a: "s5" }).url, "/api/action-items/s5")
})

test("applyResult: saved removes the op; a create stores idMap", () => {
	const c = create("a")
	const d = act("complete", { serverId: "s2" }, { completedAt: 1 })
	const dep = act("complete", { clientId: "a" }, { completedAt: 1 })
	const rec = markSent(all([c, d]), c.id)
	rec.ops.push(dep) // queued against the sent create
	const next = applyResult(rec, c, { status: "saved", serverId: "s1" })
	assert.deepEqual(next.ops.map((o) => o.id), [d.id, dep.id])
	assert.equal(next.idMap.a, "s1")
	const next2 = applyResult(applyResult(next, d, { status: "saved" }), dep, { status: "saved" })
	assert.deepEqual(next2, emptyRecord())
})

test("applyResult: gone removes the op", () => {
	const d = act("delete", { serverId: "s1" })
	assert.equal(applyResult(all([d]), d, { status: "gone" }).ops.length, 0)
})

test("applyResult: rejected flags the op, retry/auth leave it", () => {
	const d = act("delete", { serverId: "s1" })
	const rec = all([d])
	assert.equal(applyResult(rec, d, { status: "rejected" }).ops[0].rejected, true)
	assert.deepEqual(applyResult(rec, d, { status: "retry" }), rec)
	assert.deepEqual(applyResult(rec, d, { status: "auth" }), rec)
})

test("applyResult does not mutate its input", () => {
	const c = create("a")
	const rec = all([c])
	const snapshot = structuredClone(rec)
	applyResult(rec, c, { status: "saved", serverId: "s1" })
	applyResult(rec, c, { status: "rejected" })
	assert.deepEqual(rec, snapshot)
})

test("create sent, response lost (retry), then delete: create stays and a delete follows", () => {
	const c = create("a")
	let rec = markSent(all([c]), c.id)
	rec = applyResult(rec, c, { status: "retry" })
	rec = enqueue(rec, act("delete", { clientId: "a" }))
	assert.deepEqual(types(rec), ["create", "delete"])
	assert.equal(rec.ops[0].sent, true)
	// replay: create saved -> idMap resolves the delete
	rec = applyResult(rec, rec.ops[0], { status: "saved", serverId: "s1" })
	assert.deepEqual(types(rec), ["delete"])
	assert.equal(buildRequest(rec.ops[0], rec.idMap).url, "/api/action-items/s1")
})

test("delete enqueued while create is in flight, then retry: delete still queued", () => {
	const c = create("a")
	let rec = markSent(all([c]), c.id) // flush marks before fetch
	rec = enqueue(rec, act("delete", { clientId: "a" })) // user deletes mid-flight
	rec = applyResult(rec, c, { status: "retry" }) // response lost
	assert.deepEqual(types(rec), ["create", "delete"])
})

test("complete after create was sent queues a separate complete that resolves after save", () => {
	const c = create("a")
	let rec = markSent(all([c]), c.id)
	rec = enqueue(rec, act("complete", { clientId: "a" }, { completedAt: 42 }))
	assert.deepEqual(types(rec), ["create", "complete"])
	rec = applyResult(rec, c, { status: "saved", serverId: "s1" })
	assert.equal(rec.idMap.a, "s1")
	const r = buildRequest(rec.ops[0], rec.idMap)
	assert.deepEqual([r.method, r.url], ["PUT", "/api/action-items/s1"])
})

test("sent create: later complete replaces earlier complete, delete drops them", () => {
	const c = create("a")
	let rec = markSent(all([c]), c.id)
	rec = all([act("complete", { clientId: "a" }, { completedAt: 1 }), act("uncomplete", { clientId: "a" })], rec)
	assert.deepEqual(types(rec), ["create", "uncomplete"])
	rec = enqueue(rec, act("delete", { clientId: "a" }))
	assert.deepEqual(types(rec), ["create", "delete"])
})

test("markSent flags only that op and does not mutate", () => {
	const a = create("a")
	const b = create("b")
	const rec = all([a, b])
	const snapshot = structuredClone(rec)
	const next = markSent(rec, a.id)
	assert.deepEqual(rec, snapshot)
	assert.equal(next.ops[0].sent, true)
	assert.equal(next.ops[1].sent, undefined)
})

test("idMap is pruned when no op references the clientId", () => {
	const c = create("a")
	const rec = applyResult(markSent(all([c]), c.id), c, { status: "saved", serverId: "s1" })
	assert.deepEqual(rec, emptyRecord())
	// enqueue prunes too
	const stale = { ops: [], idMap: { z: "s9" } }
	assert.deepEqual(enqueue(stale, create("q")).idMap, {})
})

test("classifyResponse", () => {
	const c = create("a")
	const d = act("delete", { serverId: "s1" })
	const ok = { type: "basic", status: 200, ok: true }
	assert.deepEqual(classifyResponse(c, ok, "s1"), { status: "saved", serverId: "s1" })
	assert.deepEqual(classifyResponse(c, { ...ok, status: 201 }, ""), { status: "retry" })
	assert.deepEqual(classifyResponse(d, ok, ""), { status: "saved", serverId: undefined })
	assert.equal(classifyResponse(d, { type: "opaqueredirect", status: 0, ok: false }, "").status, "auth")
	assert.equal(classifyResponse(d, { type: "basic", status: 401, ok: false }, "").status, "auth")
	assert.equal(classifyResponse(d, { type: "basic", status: 503, ok: false }, "").status, "retry")
	assert.equal(classifyResponse(d, { type: "basic", status: 408, ok: false }, "").status, "retry")
	assert.equal(classifyResponse(d, { type: "basic", status: 429, ok: false }, "").status, "retry")
	assert.equal(classifyResponse(d, { type: "basic", status: 404, ok: false }, "").status, "gone")
	assert.equal(classifyResponse(c, { type: "basic", status: 404, ok: false }, "").status, "rejected")
	assert.equal(classifyResponse(c, { type: "basic", status: 422, ok: false }, "").status, "rejected")
})

test("dropRejected removes old rejected ops only", () => {
	const day = 86400000
	const oldRej = { ...act("delete", { serverId: "s1" }), createdAt: 0, rejected: true }
	const newRej = { ...act("delete", { serverId: "s2" }), createdAt: 9 * day, rejected: true }
	const oldOk = { ...act("delete", { serverId: "s3" }), createdAt: 0 }
	const rec = { ops: [oldRej, newRej, oldOk], idMap: {} }
	const snapshot = structuredClone(rec)
	const next = dropRejected(rec, 7 * day, 10 * day)
	assert.deepEqual(rec, snapshot)
	assert.deepEqual(next.ops.map((o) => o.ref.serverId), ["s2", "s3"])
})

test("dropRejected drops dependents of a dropped create and prunes idMap", () => {
	const day = 86400000
	const c = { ...create("a"), createdAt: 0, rejected: true, sent: true }
	const dep = act("delete", { clientId: "a" })
	const next = dropRejected({ ops: [c, dep], idMap: { a: "x" } }, 7 * day, 10 * day)
	assert.deepEqual(next, emptyRecord())
})

test("pendingCount and nextOps", () => {
	const a = act("delete", { serverId: "s1" })
	const b = act("delete", { serverId: "s2" })
	const rec = applyResult(all([a, b]), a, { status: "rejected" })
	assert.equal(pendingCount(rec), 2)
	assert.deepEqual(nextOps(rec, new Set()).map((o) => o.id), [a.id, b.id])
	assert.deepEqual(nextOps(rec, new Set([a.id])).map((o) => o.id), [b.id])
	// a non-rejected op already attempted is still eligible
	assert.deepEqual(nextOps(rec, new Set([b.id])).map((o) => o.id), [a.id, b.id])
})
