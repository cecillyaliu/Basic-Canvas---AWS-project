package svc

import (
	"context"
	"demo/database"
	"demo/metrics"
	"demo/model"
	"demo/utils"
	"encoding/json"
	//"flag"
	//"fmt"
	//"github.com/aws/aws-sdk-go-v2/config"
	//"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"time"
)

//type SNSPublishAPI interface {
//	Publish(ctx context.Context,
//		params *sns.PublishInput,
//		optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
//}

//func PublishMessage(c context.Context, api SNSPublishAPI, input *sns.PublishInput) (*sns.PublishOutput, error) {
//	return api.Publish(c, input)
//}

//func PublishSNS(msgUrl string, snsArn string) {
//	msg := flag.String("url", msgUrl, "The message to send to the subscribed users of the topic")
//	topicARN := flag.String("t", snsArn, "The ARN of the topic to which the user subscribes")
//
//	flag.Parse()
//
//	if *msg == "" || *topicARN == "" {
//		fmt.Println("You must supply a message and topic ARN")
//		fmt.Println("-m MESSAGE -t TOPIC-ARN")
//		return
//	}
//
//	cfg, err := config.LoadDefaultConfig(context.TODO())
//	if err != nil {
//		panic("configuration error, " + err.Error())
//	}
//
//	client := sns.NewFromConfig(cfg)
//
//	input := &sns.PublishInput{
//		Message:  msg,
//		TopicArn: topicARN,
//	}
//
//	result, err := PublishMessage(context.TODO(), client, input)
//	if err != nil {
//		fmt.Println("Got an error publishing the message:")
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println("Message ID: " + *result.MessageId)
//}

type Handler struct {
	statusCode int
	msg        string
}

func (d *Handler) Error(w http.ResponseWriter, msg string, statusCode int) {
	http.Error(w, msg, statusCode)
	d.statusCode = statusCode
	d.msg = msg
}

func (d *Handler) WriteHeader(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
	d.statusCode = statusCode
}

func (d *Handler) CreateAssignments(w http.ResponseWriter, r *http.Request) {
	if userInfo, exist := r.Context().Value(model.UserInfo).(*model.Account); exist {
		assignment, err := getAssignmentFromBody(r)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		assignment.ID = uuid.NewString()
		assignment.AssignmentCreated = time.Now().String()
		assignment.AssignmentUpdated = time.Now().String()
		assignment.CreatedByID = userInfo.ID
		err = database.DB.CreateAssignment(assignment)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		marshal, err := json.Marshal(assignment)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write(marshal)
		return
	}
	d.WriteHeader(w, http.StatusBadRequest)
}

func (d *Handler) GetAssignments(w http.ResponseWriter, r *http.Request) {
	var (
		assignment []*model.Assignment
		err        error
	)
	assignmentId := getAssignmentId(r.URL.Path)
	if len(assignmentId) != 0 {
		assignment, err = database.DB.GetAssignmentById(assignmentId)
	} else {
		assignment, err = database.DB.GetAllAssignment()
	}
	if err != nil {
		d.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	marshal, err := json.Marshal(assignment)
	if err != nil {
		d.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(marshal)
	if err != nil {
		return
	}
}

func (d *Handler) UpdateAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentId := getAssignmentId(r.URL.Path)
	err := d.checkPermission(r.Context(), assignmentId)
	if err != nil {
		d.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	newAssignment, err := getAssignmentFromBody(r)
	if err != nil {
		d.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newAssignment.AssignmentUpdated = time.Now().String()
	err = database.DB.UpdateAssignmentById(assignmentId, newAssignment)
	if err != nil {
		d.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.WriteHeader(w, http.StatusNoContent)

}
func (d *Handler) DeleteAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentId := getAssignmentId(r.URL.Path)
	err := d.checkPermission(r.Context(), assignmentId)
	if err != nil {
		d.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	err = database.DB.DeleteAssignmentById(assignmentId)
	if err != nil {
		d.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.WriteHeader(w, http.StatusOK)
}

func (d *Handler) Submission(w http.ResponseWriter, r *http.Request) {
	if userInfo, exist := r.Context().Value(model.UserInfo).(*model.Account); exist {
		assignmentId := getAssignmentId(r.URL.Path)
		submission, err := database.DB.GetSubmissionByAssignmentIdAndUserId(assignmentId, userInfo.ID)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		assignment, err := database.DB.GetAssignmentById(assignmentId)
		if err != nil || len(assignment) != 1 {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//check assignment exists
		now := time.Now()
		ddl, err := time.Parse("2006-01-02T15:04:05.000Z", assignment[0].Deadline)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ddl.Before(now) {
			d.Error(w, "", http.StatusForbidden)
			return
		}
		if submission.NumOfAttemps >= assignment[0].NumOfAttemps {
			d.Error(w, "", http.StatusForbidden)
			return
		}

		newSubmission, err := getSubmissionFromBody(r)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		submission.SubmissionUpdated = now.String()
		submission.SubmissionUrl = newSubmission.SubmissionUrl
		submission.NumOfAttemps += 1
		err = database.DB.UpdateSubmissionById(submission)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		marshal, err := json.Marshal(submission)
		if err != nil {
			d.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Write(marshal)
		d.WriteHeader(w, http.StatusOK)

		// Build SNS message
		message := map[string]interface{}{
			"submission_url": newSubmission.SubmissionUrl,
			"user_info": map[string]interface{}{
				"email":       userInfo.Email,
				"first_name:": userInfo.FirstName,
				"last_name":   userInfo.LastName,
			},
			"sns_arn": "",
		}

		messageJSON, err := json.Marshal(message)
		if err != nil {
			d.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create an SNS client
		sess := session.Must(session.NewSession())
		snsSvc := sns.New(sess)

		// Specify the ARN of your SNS topic
		topicArn := os.Getenv("SNS_TOPIC_ARN")
		//"arn:aws:sns:your-region:your-account-id:your-sns-topic"

		// Publish the message to the SNS topic
		_, err = snsSvc.Publish(&sns.PublishInput{
			Message:  aws.String(string(messageJSON)),
			TopicArn: aws.String(topicArn),
		})

		if err != nil {
			d.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

}

func (d *Handler) deferLog(w http.ResponseWriter, r *http.Request) {
	if d.statusCode >= 400 {
		log.Error().
			Str("ip", r.RemoteAddr).
			Str("http_method", r.Method).
			Str("path", r.URL.Path).
			Int("status_code", d.statusCode).
			Msg(d.msg)
		return
	}
	log.Info().
		Str("ip", r.RemoteAddr).
		Str("http_method", r.Method).
		Str("path", r.URL.Path).
		Int("status_code", d.statusCode).
		Msg("OK")
}

func (d *Handler) Assignments(w http.ResponseWriter, r *http.Request) {
	defer d.deferLog(w, r)

	email, password, exist := r.BasicAuth()
	if !exist {
		d.WriteHeader(w, http.StatusUnauthorized)
		return
	}
	ctx := context.Background()
	userInfo, err := database.DB.GetUserInfoByEmail(email)
	if err != nil {
		log.Printf("get pwd err :[%+v]", err)
		d.WriteHeader(w, http.StatusUnauthorized)
		return
	}
	if !utils.CheckPassword(userInfo.Password, password) {
		d.WriteHeader(w, http.StatusUnauthorized)
		return
	}
	ctx = context.WithValue(ctx, model.UserInfo, userInfo)
	r = r.WithContext(ctx)

	switch r.Method {
	case http.MethodGet:
		metrics.ThroughPut("GetAssignments")
		d.GetAssignments(w, r)
	case http.MethodPost:
		if checkIsSubmission(r.URL.Path) {
			metrics.ThroughPut("Submission")
			d.Submission(w, r)
		} else {
			metrics.ThroughPut("CreateAssignments")
			d.CreateAssignments(w, r)
		}
	case http.MethodDelete:
		metrics.ThroughPut("DeleteAssignment")
		d.DeleteAssignment(w, r)
	case http.MethodPut:
		metrics.ThroughPut("UpdateAssignment")
		d.UpdateAssignment(w, r)
	default:
		metrics.ThroughPut("Error")
		d.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func (d *Handler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	defer d.deferLog(w, r)
	if d.isDatabaseConnected() {
		d.WriteHeader(w, http.StatusOK)
	} else {
		d.WriteHeader(w, http.StatusServiceUnavailable)
	}
}

func (d *Handler) isDatabaseConnected() bool {
	if database.DB.Ping() != nil {
		return false
	}
	return true
}
