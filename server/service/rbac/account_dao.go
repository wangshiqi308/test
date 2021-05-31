/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

//Package rbac is dao layer API to help service center manage account, policy and role info
package rbac

import (
	"context"
	"fmt"
	"github.com/apache/servicecomb-service-center/datasource"
	errorsEx "github.com/apache/servicecomb-service-center/pkg/errors"
	"github.com/apache/servicecomb-service-center/pkg/log"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/plugin/quota"
	"github.com/apache/servicecomb-service-center/server/service/validator"
	"github.com/go-chassis/cari/discovery"
	"github.com/go-chassis/cari/rbac"
)

//CreateAccount save 2 kv
//1. account info
func CreateAccount(ctx context.Context, a *rbac.Account) error {
	quotaCheckErr := quota.Apply(ctx, quota.NewApplyQuotaResource(quota.TypeAccount,
		util.ParseDomainProject(ctx), "", 1))
	if quotaCheckErr != nil {
		return quotaCheckErr
	}
	err := validator.ValidateCreateAccount(a)
	if err != nil {
		log.Errorf(err, "create account [%s] failed", a.Name)
		return rbac.NewError(discovery.ErrInvalidParams, err.Error())
	}
	err = a.Check()
	if err != nil {
		log.Errorf(err, "create account [%s] failed", a.Name)
		return rbac.NewError(discovery.ErrInvalidParams, err.Error())
	}
	if err = checkRoleNames(ctx, a.Roles); err != nil {
		return rbac.NewError(rbac.ErrAccountHasInvalidRole, err.Error())
	}

	err = datasource.Instance().CreateAccount(ctx, a)
	if err == nil {
		log.Infof("create account [%s] success", a.Name)
		return nil
	}
	log.Errorf(err, "create account [%s] failed", a.Name)
	if err == datasource.ErrAccountDuplicated {
		return rbac.NewError(rbac.ErrAccountConflict, err.Error())
	}
	return err
}

// UpdateAccount updates an account's info, except the password
func UpdateAccount(ctx context.Context, name string, a *rbac.Account) error {
	// todo params validation
	if err := illegalCheck(ctx, name); err != nil {
		return err
	}
	if len(a.Status) == 0 && len(a.Roles) == 0 {
		return rbac.NewError(discovery.ErrInvalidParams, "status and roles cannot be empty both")
	}

	oldAccount, err := GetAccount(ctx, name)
	if err != nil {
		log.Errorf(err, "get account [%s] failed", name)
		return err
	}
	if len(a.Status) != 0 {
		oldAccount.Status = a.Status
	}
	if len(a.Roles) != 0 {
		oldAccount.Roles = a.Roles
	}
	if err = checkRoleNames(ctx, oldAccount.Roles); err != nil {
		return rbac.NewError(rbac.ErrAccountHasInvalidRole, err.Error())
	}
	err = datasource.Instance().UpdateAccount(ctx, name, oldAccount)
	if err != nil {
		log.Errorf(err, "can not edit account info")
		return err
	}
	log.Infof("account [%s] is edit", oldAccount.ID)
	return nil
}

func GetAccount(ctx context.Context, name string) (*rbac.Account, error) {
	r, err := datasource.Instance().GetAccount(ctx, name)
	if err != nil {
		if err == datasource.ErrAccountNotExist {
			msg := fmt.Sprintf("account [%s] not exist", name)
			return nil, rbac.NewError(rbac.ErrAccountNotExist, msg)
		}
		return nil, err
	}
	return r, nil
}
func ListAccount(ctx context.Context) ([]*rbac.Account, int64, error) {
	return datasource.Instance().ListAccount(ctx)
}
func AccountExist(ctx context.Context, name string) (bool, error) {
	return datasource.Instance().AccountExist(ctx, name)
}
func DeleteAccount(ctx context.Context, name string) error {
	if err := illegalCheck(ctx, name); err != nil {
		return err
	}
	exist, err := datasource.Instance().AccountExist(ctx, name)
	if err != nil {
		log.Errorf(err, "check account [%s] exit failed", name)
		return err
	}
	if !exist {
		msg := fmt.Sprintf("account [%s] not exist", name)
		return rbac.NewError(rbac.ErrAccountNotExist, msg)
	}
	_, err = datasource.Instance().DeleteAccount(ctx, []string{name})
	return err
}

//CreateAccount save 2 kv
//1. account info
func EditAccount(ctx context.Context, a *rbac.Account) error {
	exist, err := datasource.Instance().AccountExist(ctx, a.Name)
	if err != nil {
		log.Errorf(err, "can not edit account info")
		return err
	}
	if !exist {
		return rbac.NewError(rbac.ErrAccountNotExist, "")
	}

	err = datasource.Instance().UpdateAccount(ctx, a.Name, a)
	if err != nil {
		log.Errorf(err, "can not edit account info")
		return err
	}
	log.Infof("account [%s] is edit", a.ID)
	return nil
}

func checkRoleNames(ctx context.Context, roles []string) error {
	for _, name := range roles {
		exist, err := RoleExist(ctx, name)
		if err != nil {
			log.Errorf(err, "check role [%s] exist failed", name)
			return err
		}
		if !exist {
			return datasource.ErrRoleNotExist
		}
	}
	return nil
}

func illegalCheck(ctx context.Context, target string) error {
	if target == RootName {
		return discovery.NewError(discovery.ErrForbidden, errorsEx.MsgCantOperateRoot)
	}
	changer := UserFromContext(ctx)
	if target == changer {
		return discovery.NewError(discovery.ErrForbidden, errorsEx.MsgCantOperateYour)
	}
	return nil
}