/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package persistence

import (
	"strings"
	"testing"

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
