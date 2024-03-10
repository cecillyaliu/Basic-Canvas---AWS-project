package database

import "demo/model"

func (d *Dao) GetUserInfoByEmail(email string) (*model.Account, error) {
	res := &model.Account{}
	err := d.db.Where("email = ?", email).Take(res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}
