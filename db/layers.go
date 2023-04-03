package db

import (
	"database/sql"
	"fmt"
)

const insertLayer = `
INSERT IGNORE INTO
layers
(digest,size,instruction)
VALUES
('%s',%d,'%s')
`

func (d *DockerDB) InsertLayer(digest string, size int64, instruction string) (sql.Result, error) {

	insert := fmt.Sprintf(insertLayer,
		digest, size, EscapeString(instruction))

	return d.db.Exec(insert)
}
