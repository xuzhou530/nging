/*
   Nging is a toolbox for webmasters
   Copyright (C) 2018-present  Wenhui Shen <swh@admpub.com>

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published
   by the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package model

import (
	"strings"

	"github.com/webx-top/db"
	"github.com/webx-top/echo"

	"github.com/admpub/nging/application/dbschema"
	"github.com/admpub/nging/application/model/base"
	"github.com/admpub/nging/application/library/imbot"
	_ "github.com/admpub/nging/application/library/imbot/dingding"
	_ "github.com/admpub/nging/application/library/imbot/workwx"
)

var (
	AlertRecipientTypes = echo.NewKVData()
	AlertRecipientPlatforms = echo.NewKVData()
)

func init() {
	AlertRecipientTypes.Add(`email`, `email`)
	AlertRecipientTypes.Add(`webhook`,`webhook`)
	for name, mess := range imbot.Messagers() {
		AlertRecipientPlatforms.Add(name, mess.Label)
	}
}

func NewAlertRecipient(ctx echo.Context) *AlertRecipient {
	m := &AlertRecipient{
		NgingAlertRecipient: &dbschema.NgingAlertRecipient{},
		base:                base.New(ctx),
	}
	m.SetContext(ctx)
	return m
}

type AlertRecipient struct {
	*dbschema.NgingAlertRecipient
	base *base.Base
}

func (s *AlertRecipient) check() error {
	s.Name = strings.TrimSpace(s.Name)
	if len(s.Name) == 0 {
		return s.base.E(`名称不能为空`)
	}
	if len(s.Account) == 0 {
		return s.base.E(`账号不能为空`)
	}
	s.Description = strings.TrimSpace(s.Description)
	s.Account = strings.TrimSpace(s.Account)
	s.Type = strings.TrimSpace(s.Type)
	if len(s.Type) == 0 {
		return s.base.E(`请选择类型`)
	}
	s.Platform = strings.TrimSpace(s.Platform)
	if s.Type == `webhook` {
		if len(s.Platform) == 0 {
			return s.base.E(`对于webhook类型，必须选择一个平台`)
		}
	}
	return nil
}

func (s *AlertRecipient) GetWithExt(mw func(db.Result) db.Result, args ...interface{}) (row *AlertRecipientExt, err error) {
	err = s.NgingAlertRecipient.Get(mw, args...)
	if err != nil {
		return nil, err
	}
	row = &AlertRecipientExt{NgingAlertRecipient: s.NgingAlertRecipient}
	return row, nil
}

func (s *AlertRecipient) Add() (pk interface{}, err error) {
	if err = s.check(); err != nil {
		return nil, err
	}
	return s.NgingAlertRecipient.Add()
}

func (s *AlertRecipient) Edit(mw func(db.Result) db.Result, args ...interface{}) (err error) {
	if err = s.check(); err != nil {
		return err
	}
	return s.NgingAlertRecipient.Edit(mw, args...)
}

func (s *AlertRecipient) Send(title string, message string) (err error) {
	rows, ok := s.base.Context.Internal().Get(`NgingAlertRecipients`).([]*AlertRecipientExt)
	if !ok {
		rows = []*AlertRecipientExt{}
		_,err = s.ListByOffset(&rows, nil, 0, -1, db.Cond{`disabled`: `N`})
		if err != nil {
			return
		}
		s.base.Context.Internal().Set(`NgingAlertRecipients`, rows)
	}
	for _, row := range rows {
		row.Send(title, message)
	}
	return
}
