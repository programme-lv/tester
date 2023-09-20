package database

import (
	"github.com/jmoiron/sqlx"
)

func SelectPublishedTaskVersions(db sqlx.Queryer) (
	[]*TaskVersion, error) {
	var taskVersions []*TaskVersion

	rows, err := db.Queryx(`
		select * from task_versions
		where id in (
			select published_version_id from tasks
			where published_version_id is not null
		)`)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var version TaskVersion
		err = rows.StructScan(&version)
		if err != nil {
			return nil, err
		}
		taskVersions = append(taskVersions, &version)
	}

	return taskVersions, nil
}
