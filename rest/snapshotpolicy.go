package rest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/netapp/ontap-mcp/ontap"
	"net/http"
	"net/url"
	"regexp"
)

var intervalRegex = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)
var daysRegex = regexp.MustCompile(`(\d+)\s*day`)
var hoursRegex = regexp.MustCompile(`(\d+)\s*hour`)
var minutesRegex = regexp.MustCompile(`(\d+)\s*minute`)
var monthsRegex = regexp.MustCompile(`(\d+)\s*month`)
var weekdaysRegex = regexp.MustCompile(`(\d+)\s*weekday`)

func (c *Client) DeleteSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		ssPolicy   ontap.GetData
	)
	responseHeaders := http.Header{}
	params := url.Values{}
	params.Set("fields", "uuid")
	params.Set("name", snapshotPolicy.Name)

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&ssPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if ssPolicy.NumRecords == 0 {
		return fmt.Errorf("failed to delete snapshotPolicy=%s on svm=%s because it does not exist", snapshotPolicy.Name, snapshotPolicy.SVM.Name)
	}
	if ssPolicy.NumRecords != 1 {
		return fmt.Errorf("failed to delete snapshotPolicy=%s on svm=%s because there are %d matching records",
			snapshotPolicy.Name, snapshotPolicy.SVM.Name, ssPolicy.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/snapshot-policies/`+ssPolicy.Records[0].UUID, &statusCode, responseHeaders).
		Delete().
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if err != nil {
		return err
	}

	return c.handleJob(ctx, statusCode, buf)
}

func (c *Client) GetSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) ([]string, error) {
	var (
		ssPolicy ontap.GetData
	)
	responseHeaders := http.Header{}
	snapshotPolicies := []string{}
	params := url.Values{}
	svmName := snapshotPolicy.SVM.Name
	if svmName != "" {
		params.Set("svm", svmName)
	}

	builder := c.baseRequestBuilder(`/api/storage/snapshot-policies`, nil, responseHeaders).
		Params(params).
		ToJSON(&ssPolicy)

	err := c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return []string{}, err
	}

	if ssPolicy.NumRecords == 0 {
		if svmName != "" {
			return []string{}, fmt.Errorf("no snapshot policies found on svm: %s", svmName)
		}
		return []string{}, errors.New("no snapshot policies found in the cluster")
	}

	for _, ss := range ssPolicy.Records {
		snapshotPolicies = append(snapshotPolicies, ss.Name)
	}

	return snapshotPolicies, nil
}

func (c *Client) CreateSnapshotPolicy(ctx context.Context, snapshotPolicy ontap.SnapshotPolicy) error {
	var (
		buf        bytes.Buffer
		statusCode int
		oc         ontap.OnlyCount
		err        error
	)
	responseHeaders := http.Header{}

	// If schedule is exist then use it else create new
	scheduleName := snapshotPolicy.Copies[0].Schedule.Name
	if scheduleName == "" {
		return fmt.Errorf("no schedule exist with %s name", scheduleName)
	}
	params := url.Values{}
	params.Set("return_records", "false")
	params.Set("fields", "name")
	params.Set("name", scheduleName)

	builder := c.baseRequestBuilder(`/api/cluster/schedules`, &statusCode, responseHeaders).
		ToBytesBuffer(&buf).
		ToJSON(&oc).
		Params(params)

	err = c.buildAndExecuteRequest(ctx, builder)

	if err != nil {
		return err
	}

	if oc.NumRecords == 0 {
		// This is OK, create it
		err := c.CreateSchedule(ctx, scheduleName)
		if err != nil {
			return err
		}
	} else if oc.NumRecords != 1 {
		return fmt.Errorf("failed to create snapshotPolicy=%s on svm=%s with given schedule name=%s because there are %d matching schedules",
			snapshotPolicy.Name, snapshotPolicy.SVM.Name, scheduleName, oc.NumRecords)
	}

	builder2 := c.baseRequestBuilder(`/api/storage/snapshot-policies`, &statusCode, responseHeaders).
		BodyJSON(snapshotPolicy).
		ToBytesBuffer(&buf)

	err = c.buildAndExecuteRequest(ctx, builder2)

	if statusCode == http.StatusCreated || statusCode == http.StatusAccepted {
		return nil
	}

	return err
}

func (c *Client) CreateSchedule(ctx context.Context, scheduleName string) error {
	var statusCode int

	newSchedule, err := validateSchedule(scheduleName)
	if err != nil {
		return err
	}

	builder := c.baseRequestBuilder(`/api/cluster/schedules`, &statusCode, nil).
		BodyJSON(newSchedule)

	err = c.buildAndExecuteRequest(ctx, builder)
	if err != nil {
		return err
	}

	if statusCode != http.StatusCreated && statusCode != http.StatusAccepted {
		return fmt.Errorf(`failed to create schedule %s: unexpected status code: %d`, scheduleName, statusCode)
	}

	return nil
}

func validateSchedule(scheduleName string) (ontap.Schedule, error) {
	newSchedule := ontap.Schedule{
		Name: scheduleName,
	}

	if intervalRegex.MatchString(scheduleName) {
		fmt.Println("creating schedule using interval")
		newSchedule.Interval = scheduleName
	} else {
		matchedAny := false

		dayMatches := daysRegex.FindStringSubmatch(scheduleName)
		if len(dayMatches) > 1 {
			newSchedule.Cron.Days = append(newSchedule.Cron.Days, dayMatches[1])
			matchedAny = true
		}

		hourMatches := hoursRegex.FindStringSubmatch(scheduleName)
		if len(hourMatches) > 1 {
			newSchedule.Cron.Hours = append(newSchedule.Cron.Hours, hourMatches[1])
			matchedAny = true
		}

		minuteMatches := minutesRegex.FindStringSubmatch(scheduleName)
		if len(minuteMatches) > 1 {
			newSchedule.Cron.Minutes = append(newSchedule.Cron.Minutes, minuteMatches[1])
			matchedAny = true
		} else {
			newSchedule.Cron.Minutes = append(newSchedule.Cron.Minutes, "0")
		}

		monthMatches := monthsRegex.FindStringSubmatch(scheduleName)
		if len(monthMatches) > 1 {
			newSchedule.Cron.Months = append(newSchedule.Cron.Months, monthMatches[1])
			matchedAny = true
		}

		weekdayMatches := weekdaysRegex.FindStringSubmatch(scheduleName)
		if len(weekdayMatches) > 1 {
			newSchedule.Cron.Weekdays = append(newSchedule.Cron.Weekdays, weekdayMatches[1])
			matchedAny = true
		}

		if !matchedAny {
			return ontap.Schedule{}, fmt.Errorf("invalid schedule format: %q", scheduleName)
		}
		fmt.Println("creating schedule using cron")
	}
	return newSchedule, nil
}
