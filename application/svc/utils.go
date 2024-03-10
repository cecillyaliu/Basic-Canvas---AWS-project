package svc

import (
	"context"
	"demo/database"
	"demo/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

func getAssignmentId(path string) string {
	re := regexp.MustCompile(`/v2/assignments/([^/]+)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func checkIsSubmission(path string) bool {
	re := regexp.MustCompile(`/v2/assignments/([^/]+)/submission$`)
	matches := re.FindStringSubmatch(path)
	return len(matches) != 0
}

func getAssignmentFromBody(r *http.Request) (*model.Assignment, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("unable to read request body")
	}
	assignment := &model.Assignment{}
	err = json.Unmarshal(body, assignment)
	if err != nil {
		return nil, errors.New("json unmarshal failed")
	}
	return assignment, nil
}

func (d *Handler) checkPermission(ctx context.Context, assignmentId string) error {
	assignment, err := database.DB.GetAssignmentById(assignmentId)
	if err != nil || len(assignment) != 1 {
		return errors.New("assignment not found")
	}
	userInfo, exist := ctx.Value(model.UserInfo).(*model.Account)
	if !exist || assignment[0].CreatedByID != userInfo.ID {
		return errors.New("")
	}
	return nil
}

func getSubmissionFromBody(r *http.Request) (*model.Submission, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("unable to read request body")
	}
	submission := &model.Submission{}
	err = json.Unmarshal(body, submission)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("json unmarshal failed")
	}
	return submission, nil
}
