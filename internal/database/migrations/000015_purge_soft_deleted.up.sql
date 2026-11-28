-- Deleting now removes the row. This removes what earlier deletes only marked
-- as deleted. Destructive: restore from a backup if needed. Safe to re-run.
-- Anything still pointing at a removed row (an action item's journal, say)
-- already read as unattached, because deleted rows were never loaded.
DELETE FROM public.action_items WHERE deleted_at IS NOT NULL;
DELETE FROM public.thankfuls WHERE deleted_at IS NOT NULL;
DELETE FROM public.journals WHERE deleted_at IS NOT NULL;
DELETE FROM public.reminders WHERE deleted_at IS NOT NULL;
DELETE FROM public.entry_templates WHERE deleted_at IS NOT NULL;
DELETE FROM public.recurring_action_items WHERE deleted_at IS NOT NULL;
DELETE FROM public.custom_journal_types WHERE deleted_at IS NOT NULL;
