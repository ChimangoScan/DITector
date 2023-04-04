package crawler

import (
	"database/sql"
	"encoding/json"
)

// 实现数据持久化的简单接口

// StoreRepository__ 将Repository__直接组织成合适的形式存入数据库
func StoreRepository__(r *Repository__) (sql.Result, error) {
	var flag int8
	if r.IsPrivate {
		flag |= 1 << 0
	}
	if r.IsAutomated {
		flag |= 1 << 1
	}

	var lu, dr string
	if len(r.LastUpdated) > 19 {
		lu = r.LastUpdated[:19]
	}
	if len(r.DateRegistered) > 19 {
		dr = r.DateRegistered[:19]
	}

	return dockerDB.InsertRepository(r.User, r.Name, r.Namespace, r.RepositoryType, r.Description, flag,
		r.StarCount, r.PullCount, lu, dr, r.FullDescription)
}

// StoreTag__ 将Tag__直接组织成合适的形式存入数据库
func StoreTag__(namespace, repository string, t *Tag__) (sql.Result, error) {

	var lu, lpull, lpush string

	if len(t.LastUpdated) > 19 {
		lu = t.LastUpdated[:19]
	}
	if len(t.TagLastPulled) > 19 {
		lpull = t.TagLastPulled[:19]
	}
	if len(t.TagLastPushed) > 19 {
		lpush = t.TagLastPushed[:19]
	}

	return dockerDB.InsertTag(namespace, repository, t.Name, lu, t.LastUpdaterUsername,
		lpull, lpush, t.MediaType, t.ContentType)
}

// StoreArch__ 将Arch__组织成合适的形式存入数据库
func StoreArch__(namespace, repository, tag string, a *Arch__) (sql.Result, error) {

	b, _ := json.Marshal(a.Layers)

	var d, lpull, lpush string

	if len(a.Digest) > 8 {
		d = a.Digest[7:]
	}
	if len(a.LastPulled) > 19 {
		lpull = a.LastPulled[:19]
	}
	if len(a.LastPushed) > 19 {
		lpush = a.LastPushed[:19]
	}

	return dockerDB.InsertImage(namespace, repository, tag, a.Architecture, a.Features, a.Variant,
		d, a.OS, a.Size, a.Status, lpull, lpush, string(b))
}

// StoreLayer__ 将Layer__组织成合适的形式存入数据库
func StoreLayer__(l *Layer__) (sql.Result, error) {

	var d string

	if len(l.Digest) > 8 {
		d = l.Digest[7:]
	}

	return dockerDB.InsertLayer(d, l.Size, l.Instruction)
}
