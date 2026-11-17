package utils

// Keys for values passed between handlers, middleware, and templates via
// ctx.Locals. Use these with ctx.Locals and GetLocal instead of string literals.
// Every value is stored as a pointer (e.g. *models.User) so GetLocal[T] can read it.
const (
	LocalCurrentUser = "currentUser"

	// LocalEmailConfirmed is the *bool result of following a confirmation link.
	LocalEmailConfirmed = "emailConfirmed"

	LocalJournal                = "journal"
	LocalJournals               = "journals"
	LocalJournalTypes           = "journalTypes"
	LocalRatings                = "ratings"
	LocalOutstandingActionItems = "outstandingActionItems"

	LocalActionItem           = "actionItem"
	LocalActionItems          = "actionItems"
	LocalRecurringActionItem  = "recurringActionItem"
	LocalRecurringActionItems = "recurringActionItems"
	LocalCustomJournalType    = "customJournalType"
	LocalCustomJournalTypes   = "customJournalTypes"
	LocalThankful             = "thankful"
	LocalThankfuls            = "thankfuls"

	LocalBlog  = "blog"
	LocalBlogs = "blogs"

	// LocalReminder is the single reminder loaded by ID; LocalDailyReminder is
	// the one picked for today by SetDailyReminder.
	LocalReminder      = "reminder"
	LocalDailyReminder = "dailyReminder"
	LocalReminders     = "reminders"

	LocalEntryTemplate  = "entryTemplate"
	LocalEntryTemplates = "entryTemplates"

	LocalHasMore  = "hasMore"
	LocalNextPage = "nextPage"
	LocalPrevDate = "prevDate"
	LocalNextDate = "nextDate"

	LocalHiddenPatterns = "hiddenPatterns"

	LocalMoodChartData              = "moodChartData"
	LocalMoodByDayData              = "moodByDayData"
	LocalMoodVsActionCompletionData = "moodVsActionCompletionData"
	LocalMoodVsThankfulData         = "moodVsThankfulData"
	LocalMoodByTopicData            = "moodByTopicData"
	LocalMoodByTodData              = "moodByTodData"
	LocalFrequencyChartData         = "frequencyChartData"
	LocalThankfulFrequencyChartData = "thankfulFrequencyChartData"
	LocalRoutineCompletionRateData  = "routineCompletionRateData"
	LocalDistByTopicData            = "distByTopicData"
)
