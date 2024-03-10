package model

type Account struct {
	ID             string `gorm:"primaryKey;not null"`
	FirstName      string `gorm:"not null"`
	LastName       string `gorm:"not null"`
	Password       string `gorm:"not null"`
	Email          string `gorm:"unique;not null"`
	AccountCreated string
	AccountUpdated string
}
type Assignment struct {
	ID                string `json:"id" gorm:"primaryKey;not null"`
	Name              string `json:"name" gorm:"not null"`
	Points            int    `json:"points" gorm:"not null;check:points >= 1 AND points <= 100"`
	NumOfAttemps      int    `json:"num_of_attemps" gorm:"not null;check:num_of_attemps >= 1 AND num_of_attemps <= 100"`
	Deadline          string `json:"deadline" gorm:"not null"`
	AssignmentCreated string `json:"assignment_created"`
	AssignmentUpdated string `json:"assignment_updated"`
	CreatedByID       string `json:"-" gorm:"type:varchar(100);not null"`
}

type Submission struct {
	ID                string `json:"id" gorm:"primaryKey;not null"`
	AssignmentId      string `json:"assignment_id" gorm:"not null"`
	SubmissionUrl     string `json:"submission_url" gorm:"not null"`
	SubmissionDate    string `json:"submission_date"`
	SubmissionUpdated string `json:"submission_updated"`
	NumOfAttemps      int    `json:"-"`
	SubmitByID        string `json:"-" gorm:"type:varchar(100);not null"`
}
