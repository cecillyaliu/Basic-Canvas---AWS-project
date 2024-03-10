package database

import (
	"demo/model"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func (d *Dao) GetSubmissionByAssignmentIdAndUserId(asID, UserID string) (*model.Submission, error) {
	var res model.Submission
	err := d.db.Where("submit_by_id = ? and assignment_id = ?", UserID, asID).Take(&res).Error
	fmt.Println(err)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			//create an empty one if not exists
			res.SubmitByID = UserID
			res.AssignmentId = asID
			res.ID = uuid.NewString()
			res.SubmissionDate = time.Now().String()
			err = d.CreateSubmission(&res)
			if err != nil {
				return nil, err
			}
			return &res, nil
		}
		return nil, err
	}
	return &res, nil
}
func (d *Dao) CreateSubmission(submission *model.Submission) error {
	err := d.db.Create(submission).Error
	if err != nil {
		return err
	}
	return nil
}
func (d *Dao) UpdateSubmissionById(submission *model.Submission) error {
	return d.db.Model(&model.Submission{}).
		Where("ID = ?", submission.ID).
		Updates(submission).Error
}
