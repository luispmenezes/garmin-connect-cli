package cli

import (
	"fmt"
	"net/url"
	"time"

	"github.com/luispmenezes/garmin-connect-cli/internal/garmin"
)

type healthSpec struct {
	use         string
	short       string
	date        bool
	days        bool
	defaultDays int
	rangeFlags  bool
	post        bool
	path        func(date, from, to string, days int, args []string) (string, error)
}

func healthSpecs() []healthSpec {
	dated := func(pattern string) func(string, string, string, int, []string) (string, error) {
		return func(date, _, _ string, _ int, _ []string) (string, error) {
			return fmt.Sprintf(pattern, url.PathEscape(dateOrToday(date))), nil
		}
	}
	rangePath := func(pattern string, defaultDays int) func(string, string, string, int, []string) (string, error) {
		return func(date, from, to string, days int, _ []string) (string, error) {
			if days <= 0 {
				days = defaultDays
			}
			if to == "" {
				to = dateOrToday(date)
			}
			if from == "" {
				start, err := garmin.AddDays(to, -days+1)
				if err != nil {
					return "", err
				}
				from = start
			}
			return fmt.Sprintf(pattern, url.QueryEscape(from), url.QueryEscape(to)), nil
		}
	}
	return []healthSpec{
		{"summary", "Daily health summary", true, false, 0, false, false, func(date, _, _ string, _ int, _ []string) (string, error) {
			return fmt.Sprintf("/usersummary-service/usersummary/daily/{displayName}?calendarDate=%s", url.QueryEscape(dateOrToday(date))), nil
		}},
		{"sleep", "Sleep data", true, false, 0, false, false, func(date, _, _ string, _ int, _ []string) (string, error) {
			return fmt.Sprintf("/wellness-service/wellness/dailySleepData/{displayName}?date=%s&nonSleepBufferMinutes=60", url.QueryEscape(dateOrToday(date))), nil
		}},
		{"stress", "Stress data", true, false, 0, false, false, dated("/wellness-service/wellness/dailyStress/%s")},
		{"heart-rate", "Heart rate data", true, false, 0, false, false, func(date, _, _ string, _ int, _ []string) (string, error) {
			return fmt.Sprintf("/wellness-service/wellness/dailyHeartRate/{displayName}?date=%s", url.QueryEscape(dateOrToday(date))), nil
		}},
		{"body-battery", "Body battery data", true, true, 7, false, false, func(date, _, _ string, days int, _ []string) (string, error) {
			to := dateOrToday(date)
			from, err := garmin.AddDays(to, -days+1)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("/wellness-service/wellness/bodyBattery/reports/daily?startDate=%s&endDate=%s", from, to), nil
		}},
		{"steps", "Steps data", false, true, 7, false, false, rangePath("/usersummary-service/stats/steps/daily/%s/%s", 7)},
		{"calories", "Calories data", false, true, 7, false, false, rangePath("/usersummary-service/stats/calories/daily/%s/%s", 7)},
		{"weight", "Weight data", false, false, 0, true, false, rangePath("/weight-service/weight/dateRange?startDate=%s&endDate=%s", 30)},
		{"weight-add WEIGHT", "Add weight", true, false, 0, false, true, func(date, _, _ string, _ int, _ []string) (string, error) {
			return "/weight-service/user-weight", nil
		}},
		{"vo2max", "VO2 max", true, false, 0, false, false, dated("/metrics-service/metrics/vo2max/%s")},
		{"training-readiness", "Training readiness", true, true, 7, false, false, rangePath("/metrics-service/metrics/trainingreadiness/%s/%s", 7)},
		{"training-status", "Training status", true, true, 7, false, false, rangePath("/metrics-service/metrics/trainingstatus/%s/%s", 7)},
		{"hrv", "HRV data", true, false, 0, false, false, dated("/hrv-service/hrv/%s")},
		{"fitness-age", "Fitness age", false, false, 0, false, false, staticPath("/fitnessage-service/fitnessage")},
		{"lactate-threshold", "Lactate threshold", false, true, 90, false, false, rangePath("/metrics-service/metrics/lactatethreshold/%s/%s", 90)},
		{"race-predictions", "Race predictions", false, false, 0, false, false, staticPath("/metrics-service/metrics/racepredictions")},
		{"endurance-score", "Endurance score", false, true, 30, false, false, rangePath("/metrics-service/metrics/endurancescore/%s/%s", 30)},
		{"hill-score", "Hill score", false, true, 30, false, false, rangePath("/metrics-service/metrics/hillscore/%s/%s", 30)},
		{"spo2", "SpO2", true, false, 0, false, false, dated("/wellness-service/wellness/daily/spo2/%s")},
		{"respiration", "Respiration", true, false, 0, false, false, dated("/wellness-service/wellness/daily/respiration/%s")},
		{"intensity-minutes", "Intensity minutes", true, false, 0, false, false, dated("/wellness-service/wellness/daily/im/%s")},
		{"blood-pressure", "Blood pressure", false, false, 0, true, false, rangePath("/bloodpressure-service/bloodpressure/range/%s/%s", 30)},
		{"hydration", "Hydration", true, false, 0, false, false, dated("/usersummary-service/usersummary/hydration/%s")},
		{"personal-records", "Personal records", false, false, 0, false, false, staticPath("/personalrecord-service/personalrecord/prs")},
		{"performance-summary", "Performance summary", false, false, 0, false, false, staticPath("/metrics-service/metrics/performance/summary")},
		{"insights", "Health insights", false, true, 28, false, false, func(_ string, _ string, _ string, days int, _ []string) (string, error) {
			if days <= 0 {
				days = 28
			}
			to := time.Now().Format("2006-01-02")
			from, err := garmin.AddDays(to, -days+1)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("/insights-service/insights?startDate=%s&endDate=%s", from, to), nil
		}},
	}
}

func staticPath(path string) func(string, string, string, int, []string) (string, error) {
	return func(_, _, _ string, _ int, _ []string) (string, error) { return path, nil }
}
