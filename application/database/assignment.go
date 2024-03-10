package database

import "demo/model"

func (d *Dao) CreateAssignment(assignment *model.Assignment) error {
	err := d.db.Create(assignment).Error
	if err != nil {
		return err
	}
	return nil
}

func (d *Dao) GetAllAssignment() ([]*model.Assignment, error) {
	var res []*model.Assignment
	err := d.db.Find(&res).Error
	if err != nil {
		return res, err
	}
	return res, nil
}

func (d *Dao) GetAssignmentById(id string) ([]*model.Assignment, error) {
	var res []*model.Assignment
	err := d.db.Where("id = ?", id).Take(&res).Error
	if err != nil {
		return res, err
	}
	return res, nil
}

func (d *Dao) UpdateAssignmentById(id string, assignment *model.Assignment) error {
	return d.db.Model(&model.Assignment{}).
		Where("ID = ?", id).
		Updates(assignment).Error
}
func (d *Dao) DeleteAssignmentById(id string) error {
	return d.db.Delete(&model.Assignment{ID: id}).Error
}
