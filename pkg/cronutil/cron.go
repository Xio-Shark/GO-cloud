package cronutil

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"go-cloud/internal/domain"
)

func NextRunTime(scheduleType domain.ScheduleType, cronExpr string, runAt *time.Time, now time.Time) (*time.Time, error) {
	switch scheduleType {
	case domain.ScheduleTypeManual:
		return nil, nil
	case domain.ScheduleTypeOnce:
		if runAt == nil {
			return nil, nil
		}
		next := runAt.UTC()
		return &next, nil
	case domain.ScheduleTypeCron:
		if cronExpr == "" {
			return nil, errors.New("cron_expr is required for cron schedule")
		}
		matchers, err := parseCronExpr(cronExpr)
		if err != nil {
			return nil, err
		}
		next, err := nextCronTime(matchers, now.UTC())
		if err != nil {
			return nil, err
		}
		return &next, nil
	default:
		return nil, errors.New("unsupported schedule type")
	}
}

type cronMatcher struct {
	minute fieldMatcher
	hour   fieldMatcher
	dom    fieldMatcher
	month  fieldMatcher
	dow    fieldMatcher
}

type fieldMatcher struct {
	any    bool
	step   int
	values map[int]struct{}
}

func parseCronExpr(expr string) (cronMatcher, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return cronMatcher{}, errors.New("cron_expr must contain 5 fields")
	}
	minute, err := parseField(parts[0], 0, 59)
	if err != nil {
		return cronMatcher{}, err
	}
	hour, err := parseField(parts[1], 0, 23)
	if err != nil {
		return cronMatcher{}, err
	}
	dom, err := parseField(parts[2], 1, 31)
	if err != nil {
		return cronMatcher{}, err
	}
	month, err := parseField(parts[3], 1, 12)
	if err != nil {
		return cronMatcher{}, err
	}
	dow, err := parseField(parts[4], 0, 6)
	if err != nil {
		return cronMatcher{}, err
	}
	return cronMatcher{minute: minute, hour: hour, dom: dom, month: month, dow: dow}, nil
}

func parseField(raw string, min int, max int) (fieldMatcher, error) {
	if raw == "*" {
		return fieldMatcher{any: true}, nil
	}
	if strings.HasPrefix(raw, "*/") {
		step, err := strconv.Atoi(strings.TrimPrefix(raw, "*/"))
		if err != nil || step <= 0 {
			return fieldMatcher{}, errors.New("invalid step cron field")
		}
		return fieldMatcher{step: step}, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return fieldMatcher{}, errors.New("invalid cron field value")
	}
	return fieldMatcher{values: map[int]struct{}{value: {}}}, nil
}

func nextCronTime(matcher cronMatcher, now time.Time) (time.Time, error) {
	candidate := now.Truncate(time.Minute).Add(time.Minute)
	deadline := candidate.Add(366 * 24 * time.Hour)
	for candidate.Before(deadline) {
		if matchesTime(matcher, candidate) {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}, errors.New("unable to calculate next cron time")
}

func matchesTime(matcher cronMatcher, current time.Time) bool {
	return matchField(matcher.minute, current.Minute()) &&
		matchField(matcher.hour, current.Hour()) &&
		matchField(matcher.dom, current.Day()) &&
		matchField(matcher.month, int(current.Month())) &&
		matchField(matcher.dow, int(current.Weekday()))
}

func matchField(matcher fieldMatcher, value int) bool {
	if matcher.any {
		return true
	}
	if matcher.step > 0 {
		return value%matcher.step == 0
	}
	_, ok := matcher.values[value]
	return ok
}
