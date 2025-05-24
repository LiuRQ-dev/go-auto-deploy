package models

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DeployRecord struct {
	ID         int        `json:"id"`
	Repository string     `json:"repository"`
	Branch     string     `json:"branch"`
	Commit     string     `json:"commit"`
	Status     string     `json:"status"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Output     string     `json:"output"`
	Error      string     `json:"error,omitempty"`
}

type Database struct {
	conn *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &Database{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Database) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS deploys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repository TEXT NOT NULL,
		branch TEXT NOT NULL,
		commit TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		output TEXT,
		error TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_repository ON deploys(repository);
	CREATE INDEX IF NOT EXISTS idx_status ON deploys(status);
	CREATE INDEX IF NOT EXISTS idx_start_time ON deploys(start_time);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *Database) InsertDeploy(record *DeployRecord) (int64, error) {
	query := `
	INSERT INTO deploys (repository, branch, commit, status, start_time, output)
	VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query,
		record.Repository,
		record.Branch,
		record.Commit,
		record.Status,
		record.StartTime,
		record.Output,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *Database) UpdateDeploy(id int64, status string, endTime *time.Time, output, errorMsg string) error {
	query := `
	UPDATE deploys 
	SET status = ?, end_time = ?, output = ?, error = ?
	WHERE id = ?
	`
	_, err := db.conn.Exec(query, status, endTime, output, errorMsg, id)
	return err
}

func (db *Database) GetDeploys(limit int) ([]DeployRecord, error) {
	query := `
	SELECT id, repository, branch, commit, status, start_time, end_time, output, error
	FROM deploys
	ORDER BY start_time DESC
	LIMIT ?
	`
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []DeployRecord
	for rows.Next() {
		var record DeployRecord
		var endTime sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.Repository,
			&record.Branch,
			&record.Commit,
			&record.Status,
			&record.StartTime,
			&endTime,
			&record.Output,
			&errorMsg,
		)
		if err != nil {
			return nil, err
		}

		if endTime.Valid {
			record.EndTime = &endTime.Time
		}
		if errorMsg.Valid {
			record.Error = errorMsg.String
		}

		records = append(records, record)
	}

	return records, nil
}

func (db *Database) Close() error {
	return db.conn.Close()
}
