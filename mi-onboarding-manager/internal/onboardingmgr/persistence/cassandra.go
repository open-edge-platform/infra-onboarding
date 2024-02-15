/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/common"
	"github.com/pkg/errors"
	"github.com/scylladb/gocqlx/v2"
	"github.com/scylladb/gocqlx/v2/qb"
	"github.com/scylladb/gocqlx/v2/table"
)

var ErrInvalidKey = errors.New("Missing ID or HwID")

var _ Repository = (*Cassandra)(nil)

func getNodeTableColumns() []string {
	return common.GetFields(NodeData{})
}

func getNodeTableMetadata(keyspace string) table.Metadata {
	return table.Metadata{
		Name:    fmt.Sprintf("%s.node", keyspace),
		Columns: getNodeTableColumns(),
		PartKey: []string{"id"},
		SortKey: []string{"hwid", "fw_art_id", "os_art_id", "app_art_id", "plat_art_id"},
	}
}

func getArtifactTableColumns() []string {
	return common.GetFields(ArtifactData{})
}

func getArtifactTableMetadata(keyspace string) table.Metadata {
	return table.Metadata{
		Name:    fmt.Sprintf("%s.artifact", keyspace),
		Columns: getArtifactTableColumns(),
		PartKey: []string{"id"},
		SortKey: []string{"category", "version", "name"},
	}
}

type Cassandra struct {
	keyspace      string
	session       gocqlx.Session
	nodeTable     *table.Table
	artifactTable *table.Table
}

func newCassandra(eps []string, user, pass string, createTable bool, keyspace, replica string) (Repository, error) {
	cluster := createCluster(eps)
	if user != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: user,
			Password: pass,
		}
	}

	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		return nil, errors.Wrap(err, "create session error")
	}

	if err := createKeyspaceAndTable(session, keyspace, replica, createTable); err != nil {
		return nil, errors.Wrap(err, "createKeyspaceAndTable error")
	}

	ca := &Cassandra{
		keyspace:      keyspace,
		session:       session,
		nodeTable:     table.New(getNodeTableMetadata(keyspace)),
		artifactTable: table.New(getArtifactTableMetadata(keyspace)),
	}

	return ca, nil
}

func createCluster(clusterHosts []string, opts ...func(*gocql.ClusterConfig)) *gocql.ClusterConfig {
	cluster := gocql.NewCluster(clusterHosts...)
	const timeOutDuration, maxWaitDuration = 30 * time.Second, 2 * time.Minute
	cluster.Timeout = timeOutDuration
	cluster.Consistency = gocql.Quorum
	cluster.MaxWaitSchemaAgreement = maxWaitDuration

	for _, opt := range opts {
		opt(cluster)
	}

	return cluster
}

func (c *Cassandra) CreateNodes(ctx context.Context, data []NodeData) ([]NodeData, error) {
	qrys := make([]batchQry, 0, len(data))
	for i := range data {
		if data[i].HwID == "" {
			return nil, ErrInvalidKey
		}
		id := gocql.TimeUUID()
		data[i].ID = id.String()
		qrys = append(qrys, nodeInsertQry(c.keyspace, data[i]))
	}
	b := c.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for _, qry := range qrys {
		b.Query(qry.qry, qry.arg...)
	}
	if err := c.session.ExecuteBatch(b); err != nil {
		return nil, errors.Wrap(err, "CreateNodes batch execution")
	}
	return data, nil
}

func (c *Cassandra) UpdateNodes(ctx context.Context, data []NodeData) error {
	qrys := make([]batchQry, 0, len(data))
	for _, d := range data {
		if d.HwID == "" && d.ID == "" {
			return ErrInvalidKey
		}
		qrys = append(qrys, nodeUpdateQry(c.keyspace, d))
	}
	b := c.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for _, qry := range qrys {
		b.Query(qry.qry, qry.arg...)
	}
	if err := c.session.ExecuteBatch(b); err != nil {
		return errors.Wrap(err, "UpdateNodes batch execution")
	}
	return nil
}

func (c *Cassandra) GetNodes(_ context.Context, data NodeData) ([]*NodeData, error) {
	get := c.session.Query(nodeSelQry(c.keyspace, data))
	get.BindStruct(data)
	var items []*NodeData
	if err := get.SelectRelease(&items); err != nil {
		return nil, errors.Wrap(err, "GetNodes")
	}
	return items, nil
}

func (c *Cassandra) DeleteNodes(ctx context.Context, ids []string) error {
	qrys := make([]batchQry, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			return ErrInvalidKey
		}
		qrys = append(qrys, nodeDelQry(c.keyspace, NodeData{ID: id}))
	}
	b := c.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for _, qry := range qrys {
		b.Query(qry.qry, qry.arg...)
	}
	if err := c.session.ExecuteBatch(b); err != nil {
		return errors.Wrap(err, "DeleteNodes batch execution")
	}
	return nil
}

func (c *Cassandra) CreateArtifacts(ctx context.Context, data []ArtifactData) ([]ArtifactData, error) {
	qrys := make([]batchQry, 0, len(data))
	for i := range data {
		id := gocql.TimeUUID()
		data[i].ID = id.String()
		qrys = append(qrys, artifactInsertQry(c.keyspace, data[i]))
	}
	b := c.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for _, qry := range qrys {
		b.Query(qry.qry, qry.arg...)
	}
	if err := c.session.ExecuteBatch(b); err != nil {
		return nil, errors.Wrap(err, "CreateArtifacts batch execution")
	}
	return data, nil
}

func (c *Cassandra) UpdateArtifacts(ctx context.Context, data []ArtifactData) error {
	qrys := make([]batchQry, 0, len(data))
	for _, d := range data {
		if d.ID == "" {
			return ErrInvalidKey
		}
		qrys = append(qrys, artifactUpdateQry(c.keyspace, d))
	}
	b := c.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for _, qry := range qrys {
		b.Query(qry.qry, qry.arg...)
	}
	if err := c.session.ExecuteBatch(b); err != nil {
		return errors.Wrap(err, "UpdateArtifacts batch execution")
	}
	return nil
}

func (c *Cassandra) GetArtifacts(_ context.Context, data ArtifactData) ([]*ArtifactData, error) {
	get := c.session.Query(artifactSelQry(c.keyspace, data))
	get.BindStruct(data)
	var items []*ArtifactData
	if err := get.SelectRelease(&items); err != nil {
		return nil, errors.Wrap(err, "GetArtifacts")
	}
	return items, nil
}

func (c *Cassandra) DeleteArtifacts(ctx context.Context, ids []string) error {
	qrys := make([]batchQry, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			return ErrInvalidKey
		}
		qrys = append(qrys, artifactDelQry(c.keyspace, ArtifactData{ID: id}))
	}
	b := c.session.NewBatch(gocql.LoggedBatch).WithContext(ctx)
	for _, qry := range qrys {
		b.Query(qry.qry, qry.arg...)
	}
	if err := c.session.ExecuteBatch(b); err != nil {
		return errors.Wrap(err, "DeleteArtifacts batch execution")
	}
	return nil
}

func (c *Cassandra) Close() error {
	c.session.Close()
	return nil
}

func createKeyspaceAndTable(session gocqlx.Session, keyspace, replica string, createTable bool) error {
	err := session.ExecStmt(fmt.Sprintf(
		`CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor': %s}`,
		keyspace, replica,
	))
	if err != nil {
		return errors.Wrap(err, "create keyspace")
	}

	if createTable {
		if err = session.ExecStmt(fmt.Sprintf("DROP TABLE IF EXISTS  %s.node", keyspace)); err != nil {
			return errors.Wrap(err, "drop table")
		}

		if err = session.ExecStmt(fmt.Sprintf("DROP TABLE IF EXISTS  %s.artifact", keyspace)); err != nil {
			return errors.Wrap(err, "drop table")
		}

		if err = session.ExecStmt(fmt.Sprintf("DROP TABLE IF EXISTS  %s.profile", keyspace)); err != nil {
			return errors.Wrap(err, "drop table")
		}

		if err = session.ExecStmt(fmt.Sprintf("DROP TABLE IF EXISTS  %s.group", keyspace)); err != nil {
			return errors.Wrap(err, "drop table")
		}
	}

	err = session.ExecStmt(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.node (
		id text PRIMARY KEY,
		hwid text,
		plat_type text,
		fw_art_id text,
		os_art_id text,
		app_art_id text,
		plat_art_id text,
		dev_type text,
		dev_info_agent text,
		dev_status text,
		onboard_status text,
		update_status text,
		update_avl text)`, keyspace))
	if err != nil {
		return errors.Wrap(err, "create table")
	}

	err = session.ExecStmt(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.artifact (
		id text PRIMARY KEY,
		category text,
		name text,
		version text,
		descrip text,
		detail text,
		pkg_url text,
		author text,
		state text,
		license text)`, keyspace))
	if err != nil {
		return errors.Wrap(err, "create table")
	}

	err = session.ExecStmt(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.profile (
		id text PRIMARY KEY,
		name text,
		onboard_params text,
		customer_params text,
		start_onboard boolean,
		os_art_id text,
		fw_art_id text,
		img_art_id text,
		app_art_id text,
		hw_data text)`, keyspace))
	if err != nil {
		return errors.Wrap(err, "create table")
	}

	err = session.ExecStmt(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.group (
		id text PRIMARY KEY,
		name text,
		node_ids text)`, keyspace))
	if err != nil {
		return errors.Wrap(err, "create table")
	}

	return nil
}

type batchQry struct {
	qry string
	arg []interface{}
}

func nodeInsertQry(keyspace string, data NodeData) batchQry {
	qry, _ := qb.Insert(fmt.Sprintf("%s.node", keyspace)).
		Columns(getNodeTableColumns()...).ToCql()
	arg := common.GetValues(data)
	return batchQry{qry, arg}
}

func nodeUpdateQry(keyspace string, data NodeData) batchQry {
	c := getNodeTableColumns()
	for i := 0; i < len(c); i++ {
		// can NOT update id / hwid column
		if c[i] == "id" || c[i] == "hwid" {
			c[i] = c[len(c)-1]
			c = c[:len(c)-1]
		}
	}

	qry, _ := qb.Update(fmt.Sprintf("%s.node", keyspace)).
		Set(c...).Where(qb.Eq("id")).ToCql()
	vv := common.GetMapValues(data)
	arg := make([]interface{}, 0, len(vv))
	for _, k := range c {
		if v, ok := vv[k]; ok {
			arg = append(arg, v)
		}
	}
	if v, ok := vv["id"]; ok {
		arg = append(arg, v)
	}

	return batchQry{qry, arg}
}

func nodeDelQry(keyspace string, data NodeData) batchQry {
	qry, _ := qb.Delete(fmt.Sprintf("%s.node", keyspace)).
		Where(qb.Eq("id")).ToCql()
	arg := []interface{}{data.ID}
	return batchQry{qry, arg}
}

func nodeSelQry(keyspace string, data NodeData) (stmt string, names []string) {
	f, v := common.GetFields(data), common.GetValues(data)
	w := make([]qb.Cmp, 0, len(v))
	for i := range v {
		u, ok := v[i].(string)
		n, ko := v[i].(fmt.Stringer)
		if (ok && u != "") || (ko && n.String() != "") {
			w = append(w, qb.Eq(f[i]))
		}
	}
	return qb.Select(fmt.Sprintf("%s.node", keyspace)).
		Where(w...).AllowFiltering().ToCql()
}

func artifactInsertQry(keyspace string, data ArtifactData) batchQry {
	qry, _ := qb.Insert(fmt.Sprintf("%s.artifact", keyspace)).
		Columns(getArtifactTableColumns()...).ToCql()
	arg := common.GetValues(data)
	return batchQry{qry, arg}
}

func artifactUpdateQry(keyspace string, data ArtifactData) batchQry {
	c := getArtifactTableColumns()
	for i := 0; i < len(c); i++ {
		// can NOT update id column
		if c[i] == "id" {
			c[i] = c[len(c)-1]
			c = c[:len(c)-1]
		}
	}

	qry, _ := qb.Update(fmt.Sprintf("%s.artifact", keyspace)).
		Set(c...).Where(qb.Eq("id")).ToCql()
	vv := common.GetMapValues(data)
	arg := make([]interface{}, 0, len(vv))
	for _, k := range c {
		if v, ok := vv[k]; ok {
			arg = append(arg, v)
		}
	}
	if v, ok := vv["id"]; ok {
		arg = append(arg, v)
	}

	return batchQry{qry, arg}
}

func artifactDelQry(keyspace string, data ArtifactData) batchQry {
	qry, _ := qb.Delete(fmt.Sprintf("%s.artifact", keyspace)).
		Where(qb.Eq("id")).ToCql()
	arg := []interface{}{data.ID}
	return batchQry{qry, arg}
}

func artifactSelQry(keyspace string, data ArtifactData) (stmt string, names []string) {
	f, v := common.GetFields(data), common.GetValues(data)
	w := make([]qb.Cmp, 0, len(v))
	for i := range v {
		u, ok := v[i].(string)
		n, ko := v[i].(fmt.Stringer)
		if (ok && u != "") || (ko && n.String() != "") {
			w = append(w, qb.Eq(f[i]))
		}
	}
	return qb.Select(fmt.Sprintf("%s.artifact", keyspace)).
		Where(w...).AllowFiltering().ToCql()
}
