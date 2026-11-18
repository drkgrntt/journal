package utils

// Keys for values passed between handlers, middleware, and templates via
// ctx.Locals. Use these with ctx.Locals and GetLocal instead of string literals.
// Every value is stored as a pointer (e.g. *models.User) so GetLocal[T] can read it.
const (
	LocalCurrentUser = "currentUser"

	// LocalRequestID is the *string random reference for this request (set by
	// middleware.RequestID); LocalRequestError is the *string safe description of
	// why a request failed (see SetRequestError).
	LocalRequestID    = "requestID"
	LocalRequestError = "requestError"

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

	// LocalCalendarCounts is the *map[string]int of per-day counts ("2006-01-02")
	// for the calendar's non-rating lenses.
	LocalCalendarCounts = "calendarCounts"

	LocalMoodChartData              = "moodChartData"
	LocalYearInPixels               = "yearInPixels"
	LocalMoodVsActionCompletionData = "moodVsActionCompletionData"
	LocalMoodVsThankfulData         = "moodVsThankfulData"
	LocalMoodByTopicData            = "moodByTopicData"
	LocalWeekByTimeData             = "weekByTimeData"
	LocalRatingDistributionData     = "ratingDistributionData"
	LocalRoutineCompletionRateData  = "routineCompletionRateData"
	LocalDistByTopicData            = "distByTopicData"
)
