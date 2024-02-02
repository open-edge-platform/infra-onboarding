/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package persistence

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/scylladb/gocqlx/v2"
	"github.com/scylladb/gocqlx/v2/table"
	"github.com/stretchr/testify/assert"
)

func TestNodeSelect(t *testing.T) {
	data := NodeData{DeviceType: "test"}
	keyspace := "key"
	stmt, names := nodeSelQry(keyspace, data)
	stmtExp := "SELECT * FROM key.node WHERE dev_type=? ALLOW FILTERING"
	namesExp := []string{"dev_type"}
	assert.Equal(t, stmtExp, strings.TrimSpace(stmt))
	assert.Equal(t, namesExp, names)
}

func TestNodeInsert(t *testing.T) {
	data := NodeData{HwID: "xxx", DeviceType: "test"}
	keyspace := "key"
	qry := nodeInsertQry(keyspace, data)
	stmtExp := "INSERT INTO key.node (id,hwid,plat_type,fw_art_id,os_art_id,app_art_id,plat_art_id,dev_type,dev_info_agent,dev_status,update_status,update_avl,onboard_status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)"
	assert.Equal(t, stmtExp, strings.TrimSpace(qry.qry))
}

func TestNodeUpdate(t *testing.T) {
	data := NodeData{HwID: "xxx", DeviceType: "test"}
	keyspace := "key"
	qry := nodeUpdateQry(keyspace, data)
	stmtExp := "UPDATE key.node SET onboard_status=?,update_avl=?,plat_type=?,fw_art_id=?,os_art_id=?,app_art_id=?,plat_art_id=?,dev_type=?,dev_info_agent=?,dev_status=?,update_status=? WHERE id=?"
	assert.Equal(t, stmtExp, strings.TrimSpace(qry.qry))
}

func TestNodeDelete(t *testing.T) {
	data := NodeData{ID: "xxx", HwID: "xxx", DeviceType: "test"}
	keyspace := "key"
	qry := nodeDelQry(keyspace, data)
	stmtExp := "DELETE FROM key.node WHERE id=?"
	assert.Equal(t, stmtExp, strings.TrimSpace(qry.qry))
}

func TestArtifactSelect(t *testing.T) {
	data := ArtifactData{Category: Platform, Name: "xxx"}
	keyspace := "key"
	stmt, names := artifactSelQry(keyspace, data)
	stmtExp := "SELECT * FROM key.artifact WHERE category=? AND name=? ALLOW FILTERING"
	namesExp := []string{"category", "name"}
	assert.Equal(t, stmtExp, strings.TrimSpace(stmt))
	assert.Equal(t, namesExp, names)
}

func TestArtifactInsert(t *testing.T) {
	data := ArtifactData{Category: Platform, Name: "xxx"}
	keyspace := "key"
	qry := artifactInsertQry(keyspace, data)
	stmtExp := "INSERT INTO key.artifact (id,category,name,version,descrip,detail,pkg_url,author,state,license) VALUES (?,?,?,?,?,?,?,?,?,?)"
	assert.Equal(t, stmtExp, strings.TrimSpace(qry.qry))
}

func TestArtifactUpdate(t *testing.T) {
	data := ArtifactData{Category: Platform, Name: "xxx"}
	keyspace := "key"
	qry := artifactUpdateQry(keyspace, data)
	stmtExp := "UPDATE key.artifact SET license=?,category=?,name=?,version=?,descrip=?,detail=?,pkg_url=?,author=?,state=? WHERE id=?"
	assert.Equal(t, stmtExp, strings.TrimSpace(qry.qry))
}

func TestArtifactDelete(t *testing.T) {
	data := ArtifactData{Category: Platform, Name: "xxx"}
	keyspace := "key"
	qry := artifactDelQry(keyspace, data)
	stmtExp := "DELETE FROM key.artifact WHERE id=?"
	assert.Equal(t, stmtExp, strings.TrimSpace(qry.qry))
}

func TestCassandra_CreateNodes(t *testing.T) {
	type fields struct {
		keyspace      string
		session       gocqlx.Session
		nodeTable     *table.Table
		artifactTable *table.Table
	}
	type args struct {
		ctx  context.Context
		data []NodeData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []NodeData
		wantErr bool
	}{
		{
			name:   "TestCase1",
			fields: fields{},
			args: args{
				ctx: context.Background(),
				data: []NodeData{{
					HwID: "",
				}},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cassandra{
				keyspace:      tt.fields.keyspace,
				session:       tt.fields.session,
				nodeTable:     tt.fields.nodeTable,
				artifactTable: tt.fields.artifactTable,
			}
			got, err := c.CreateNodes(tt.args.ctx, tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cassandra.CreateNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Cassandra.CreateNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCassandra_UpdateNodes(t *testing.T) {
	type fields struct {
		keyspace      string
		session       gocqlx.Session
		nodeTable     *table.Table
		artifactTable *table.Table
	}
	type args struct {
		ctx  context.Context
		data []NodeData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestCase1",
			fields: fields{},
			args: args{
				ctx: context.Background(),
				data: []NodeData{{
					HwID: "",
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cassandra{
				keyspace:      tt.fields.keyspace,
				session:       tt.fields.session,
				nodeTable:     tt.fields.nodeTable,
				artifactTable: tt.fields.artifactTable,
			}
			if err := c.UpdateNodes(tt.args.ctx, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Cassandra.UpdateNodes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCassandra_DeleteNodes(t *testing.T) {
	type fields struct {
		keyspace      string
		session       gocqlx.Session
		nodeTable     *table.Table
		artifactTable *table.Table
	}
	type args struct {
		ctx context.Context
		ids []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestCase1",
			fields: fields{},
			args: args{
				ctx: context.Background(),
				ids: []string{""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cassandra{
				keyspace:      tt.fields.keyspace,
				session:       tt.fields.session,
				nodeTable:     tt.fields.nodeTable,
				artifactTable: tt.fields.artifactTable,
			}
			if err := c.DeleteNodes(tt.args.ctx, tt.args.ids); (err != nil) != tt.wantErr {
				t.Errorf("Cassandra.DeleteNodes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCassandra_UpdateArtifacts(t *testing.T) {
	type fields struct {
		keyspace      string
		session       gocqlx.Session
		nodeTable     *table.Table
		artifactTable *table.Table
	}
	type args struct {
		ctx  context.Context
		data []ArtifactData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestCase1",
			fields: fields{},
			args: args{
				ctx:  context.Background(),
				data: []ArtifactData{{ID: ""}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cassandra{
				keyspace:      tt.fields.keyspace,
				session:       tt.fields.session,
				nodeTable:     tt.fields.nodeTable,
				artifactTable: tt.fields.artifactTable,
			}
			if err := c.UpdateArtifacts(tt.args.ctx, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Cassandra.UpdateArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCassandra_DeleteArtifacts(t *testing.T) {
	type fields struct {
		keyspace      string
		session       gocqlx.Session
		nodeTable     *table.Table
		artifactTable *table.Table
	}
	type args struct {
		ctx context.Context
		ids []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestCase1",
			fields: fields{},
			args: args{
				ctx: context.Background(),
				ids: []string{""},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Cassandra{
				keyspace:      tt.fields.keyspace,
				session:       tt.fields.session,
				nodeTable:     tt.fields.nodeTable,
				artifactTable: tt.fields.artifactTable,
			}
			if err := c.DeleteArtifacts(tt.args.ctx, tt.args.ids); (err != nil) != tt.wantErr {
				t.Errorf("Cassandra.DeleteArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getNodeTableMetadata(t *testing.T) {
	type args struct {
		keyspace string
	}
	tests := []struct {
		name string
		args args
		want table.Metadata
	}{
		{
			name: "Test Case 1",
			args: args{},
			want: table.Metadata{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNodeTableMetadata(tt.args.keyspace); reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNodeTableMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getArtifactTableMetadata(t *testing.T) {
	type args struct {
		keyspace string
	}
	tests := []struct {
		name string
		args args
		want table.Metadata
	}{
		{
			name: "Test Case",
			args: args{},
			want: table.Metadata{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getArtifactTableMetadata(tt.args.keyspace); reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArtifactTableMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newCassandra(t *testing.T) {
	type args struct {
		eps         []string
		user        string
		pass        string
		createTable bool
		keyspace    string
		replica     string
	}
	tests := []struct {
		name    string
		args    args
		want    Repository
		wantErr bool
	}{
		{
			name:    "Test Case 1",
			args:    args{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Test Case 2",
			args:    args{user: "abc"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Test Case 2",
			args:    args{user: "abc", pass: "pass", eps: []string{"addr"}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newCassandra(tt.args.eps, tt.args.user, tt.args.pass, tt.args.createTable, tt.args.keyspace, tt.args.replica)
			if (err != nil) != tt.wantErr {
				t.Errorf("newCassandra() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newCassandra() = %v, want %v", got, tt.want)
			}
		})
	}
}
