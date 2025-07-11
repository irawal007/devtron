/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package configure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	apiBean "github.com/devtron-labs/devtron/api/restHandler/app/pipeline/configure/bean"
	"github.com/devtron-labs/devtron/internal/sql/constants"
	"github.com/devtron-labs/devtron/pkg/build/artifacts/imageTagging"
	bean2 "github.com/devtron-labs/devtron/pkg/build/pipeline/bean"
	eventProcessorBean "github.com/devtron-labs/devtron/pkg/eventProcessor/bean"
	constants2 "github.com/devtron-labs/devtron/pkg/pipeline/constants"
	"github.com/devtron-labs/devtron/util/stringsUtil"
	"golang.org/x/exp/maps"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/devtron-labs/devtron/util/response/pagination"
	"github.com/gorilla/schema"

	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/client/gitSensor"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	dockerRegistryRepository "github.com/devtron-labs/devtron/internal/sql/repository/dockerRegistry"
	"github.com/devtron-labs/devtron/internal/sql/repository/helper"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/bean"
	bean1 "github.com/devtron-labs/devtron/pkg/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/types"
	resourceGroup "github.com/devtron-labs/devtron/pkg/resourceGroup"
	"github.com/devtron-labs/devtron/util/response"
	"github.com/go-pg/pg"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
)

type DevtronAppBuildRestHandler interface {
	CreateCiConfig(w http.ResponseWriter, r *http.Request)
	UpdateCiTemplate(w http.ResponseWriter, r *http.Request)

	GetCiPipeline(w http.ResponseWriter, r *http.Request)
	GetExternalCi(w http.ResponseWriter, r *http.Request)
	GetExternalCiById(w http.ResponseWriter, r *http.Request)
	PatchCiPipelines(w http.ResponseWriter, r *http.Request)
	PatchCiMaterialSourceWithAppIdAndEnvironmentId(w http.ResponseWriter, r *http.Request)
	PatchCiMaterialSourceWithAppIdsAndEnvironmentId(w http.ResponseWriter, r *http.Request)
	TriggerCiPipeline(w http.ResponseWriter, r *http.Request)
	GetCiPipelineMin(w http.ResponseWriter, r *http.Request)
	GetCIPipelineById(w http.ResponseWriter, r *http.Request)
	GetCIPipelineByPipelineId(w http.ResponseWriter, r *http.Request)
	HandleWorkflowWebhook(w http.ResponseWriter, r *http.Request)
	GetBuildLogs(w http.ResponseWriter, r *http.Request)
	FetchWorkflowDetails(w http.ResponseWriter, r *http.Request)
	GetArtifactsForCiJob(w http.ResponseWriter, r *http.Request)
	// CancelWorkflow CancelBuild
	CancelWorkflow(w http.ResponseWriter, r *http.Request)

	UpdateBranchCiPipelinesWithRegex(w http.ResponseWriter, r *http.Request)
	GetAppMetadataListByEnvironment(w http.ResponseWriter, r *http.Request)
	GetCiPipelineByEnvironment(w http.ResponseWriter, r *http.Request)
	GetCiPipelineByEnvironmentMin(w http.ResponseWriter, r *http.Request)
	GetExternalCiByEnvironment(w http.ResponseWriter, r *http.Request)
	// GetSourceCiDownStreamFilters will fetch the environments attached to all the linked CIs for the given ciPipelineId
	GetSourceCiDownStreamFilters(w http.ResponseWriter, r *http.Request)
	// GetSourceCiDownStreamInfo will fetch the deployment information of all the linked CIs for the given ciPipelineId
	GetSourceCiDownStreamInfo(w http.ResponseWriter, r *http.Request)
}

type DevtronAppBuildMaterialRestHandler interface {
	CreateMaterial(w http.ResponseWriter, r *http.Request)
	UpdateMaterial(w http.ResponseWriter, r *http.Request)
	FetchMaterials(w http.ResponseWriter, r *http.Request)
	FetchMaterialsByMaterialId(w http.ResponseWriter, r *http.Request)
	RefreshMaterials(w http.ResponseWriter, r *http.Request)
	FetchMaterialInfo(w http.ResponseWriter, r *http.Request)
	FetchChanges(w http.ResponseWriter, r *http.Request)
	DeleteMaterial(w http.ResponseWriter, r *http.Request)
	GetCommitMetadataForPipelineMaterial(w http.ResponseWriter, r *http.Request)
}

type DevtronAppBuildHistoryRestHandler interface {
	GetHistoricBuildLogs(w http.ResponseWriter, r *http.Request)
	GetBuildHistory(w http.ResponseWriter, r *http.Request)
	DownloadCiWorkflowArtifacts(w http.ResponseWriter, r *http.Request)
}

type ImageTaggingRestHandler interface {
	CreateUpdateImageTagging(w http.ResponseWriter, r *http.Request)
	GetImageTaggingData(w http.ResponseWriter, r *http.Request)
}

func (handler *PipelineConfigRestHandlerImpl) CreateCiConfig(w http.ResponseWriter, r *http.Request) {
	userId, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	var createRequest bean.CiConfigRequest
	if !handler.decodeJsonBody(w, r, &createRequest, "create ci config") {
		return
	}
	createRequest.UserId = userId

	handler.Logger.Infow("request payload, create ci config", "create request", createRequest)

	if !handler.validateRequestBody(w, createRequest, "create ci config") {
		return
	}

	// validates if the dockerRegistry can store CONTAINER
	isValid := handler.dockerRegistryConfig.ValidateRegistryStorageType(createRequest.DockerRegistry, dockerRegistryRepository.OCI_REGISRTY_REPO_TYPE_CONTAINER, dockerRegistryRepository.STORAGE_ACTION_TYPE_PUSH, dockerRegistryRepository.STORAGE_ACTION_TYPE_PULL_AND_PUSH)
	if !isValid {
		err := fmt.Errorf("invalid registry type")
		handler.Logger.Errorw("validation err, create ci config", "err", err, "create request", createRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	token := r.Header.Get("token")
	_, authorized := handler.getAppAndCheckAuthForAction(w, createRequest.AppId, token, casbin.ActionCreate)
	if !authorized {
		return
	}

	createResp, err := handler.pipelineBuilder.CreateCiPipeline(&createRequest)
	if err != nil {
		handler.Logger.Errorw("service err, create", "err", err, "create request", createRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) UpdateCiTemplate(w http.ResponseWriter, r *http.Request) {
	userId, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	var configRequest bean.CiConfigRequest
	if !handler.decodeJsonBody(w, r, &configRequest, "UpdateCiTemplate") {
		return
	}
	configRequest.UserId = userId

	handler.Logger.Infow("request payload, update ci template", "UpdateCiTemplate", configRequest, "userId", userId)

	if !handler.validateRequestBody(w, configRequest, "UpdateCiTemplate") {
		return
	}

	token := r.Header.Get("token")
	_, authorized := handler.getAppAndCheckAuthForAction(w, configRequest.AppId, token, casbin.ActionCreate)
	if !authorized {
		return
	}

	createResp, err := handler.pipelineBuilder.UpdateCiTemplate(&configRequest)
	if err != nil {
		handler.Logger.Errorw("service err, UpdateCiTemplate", "UpdateCiTemplate", configRequest, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) UpdateBranchCiPipelinesWithRegex(w http.ResponseWriter, r *http.Request) {
	userId, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	var patchRequest bean.CiRegexPatchRequest
	if !handler.decodeJsonBody(w, r, &patchRequest, "PatchCiPipelines") {
		return
	}
	patchRequest.UserId = userId

	handler.Logger.Debugw("update request ", "req", patchRequest)

	token := r.Header.Get("token")
	_, authorized := handler.getAppAndCheckAuthForAction(w, patchRequest.AppId, token, casbin.ActionTrigger)
	if !authorized {
		return
	}

	// Filter materials that have regex configured
	var materialList []*bean.CiPipelineMaterial
	for _, material := range patchRequest.CiPipelineMaterial {
		if handler.ciPipelineMaterialRepository.CheckRegexExistsForMaterial(material.Id) {
			materialList = append(materialList, material)
		}
	}
	if len(materialList) == 0 {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	patchRequest.CiPipelineMaterial = materialList

	// Update the pipeline
	err := handler.pipelineBuilder.PatchRegexCiPipeline(&patchRequest)
	if err != nil {
		handler.Logger.Errorw("service err, PatchCiPipelines", "err", err, "PatchCiPipelines", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	//if include/exclude configured showAll will include excluded materials also in list, if not configured it will ignore this flag
	resp, err := handler.ciHandler.FetchMaterialsByPipelineId(patchRequest.Id, false)
	if err != nil {
		handler.Logger.Errorw("service err, FetchMaterials", "pipelineId", patchRequest.Id, "err", err)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) parseSourceChangeRequest(w http.ResponseWriter, r *http.Request) (*bean.CiMaterialPatchRequest, int32, error) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return nil, 0, err
	}

	var patchRequest bean.CiMaterialPatchRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&patchRequest)
	if err != nil {
		handler.Logger.Errorw("request err, PatchCiPipeline", "err", err, "PatchCiPipeline", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return nil, 0, err
	}

	return &patchRequest, userId, nil
}

func (handler *PipelineConfigRestHandlerImpl) parseBulkSourceChangeRequest(w http.ResponseWriter, r *http.Request) (*bean.CiMaterialBulkPatchRequest, int32, error) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return nil, 0, err
	}

	var patchRequest bean.CiMaterialBulkPatchRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&patchRequest)
	if err != nil {
		handler.Logger.Errorw("request err, BulkPatchCiPipeline", "err", err, "BulkPatchCiPipeline", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return nil, 0, err
	}

	err = handler.validator.Struct(patchRequest)
	if err != nil {
		handler.Logger.Errorw("request err, BulkPatchCiPipeline", "BulkPatchCiPipeline", patchRequest, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return nil, 0, err
	}

	return &patchRequest, userId, nil
}

func (handler *PipelineConfigRestHandlerImpl) authorizeCiSourceChangeRequest(w http.ResponseWriter, patchRequest *bean.CiMaterialPatchRequest, token string) error {
	handler.Logger.Debugw("update request ", "req", patchRequest)
	app, err := handler.pipelineBuilder.GetApp(patchRequest.AppId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return err
	}
	if app.AppType != helper.CustomApp {
		err = fmt.Errorf("only custom apps supported")
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return err
	}
	resourceName := handler.enforcerUtil.GetAppRBACName(app.AppName)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionUpdate, resourceName); !ok {
		err = fmt.Errorf("unauthorized user")
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return err
	}
	err = handler.validator.Struct(patchRequest)
	if err != nil {
		handler.Logger.Errorw("validation err", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return err
	}
	return nil
}

func (handler *PipelineConfigRestHandlerImpl) PatchCiMaterialSourceWithAppIdAndEnvironmentId(w http.ResponseWriter, r *http.Request) {
	patchRequest, userId, err := handler.parseSourceChangeRequest(w, r)
	if err != nil {
		handler.Logger.Errorw("Parse error, PatchCiMaterialSource", "err", err, "PatchCiMaterialSource", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	if !(patchRequest.Source.Type == constants.SOURCE_TYPE_BRANCH_FIXED || patchRequest.Source.Type == constants.SOURCE_TYPE_BRANCH_REGEX) {
		handler.Logger.Errorw("Unsupported source type, PatchCiMaterialSource", "err", err, "PatchCiMaterialSource", patchRequest)
		common.WriteJsonResp(w, err, "source.type not supported", http.StatusBadRequest)
		return
	}
	token := r.Header.Get("token")
	if err = handler.authorizeCiSourceChangeRequest(w, patchRequest, token); err != nil {
		handler.Logger.Errorw("Authorization error, PatchCiMaterialSource", "err", err, "PatchCiMaterialSource", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusUnauthorized)
		return
	}

	createResp, err := handler.pipelineBuilder.PatchCiMaterialSource(patchRequest, userId)
	if err != nil {
		handler.Logger.Errorw("service err, PatchCiPipelines", "err", err, "PatchCiPipelines", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) PatchCiMaterialSourceWithAppIdsAndEnvironmentId(w http.ResponseWriter, r *http.Request) {
	bulkPatchRequest, userId, err := handler.parseBulkSourceChangeRequest(w, r)
	if err != nil {
		handler.Logger.Errorw("Parse error, PatchCiMaterialSource", "err", err, "PatchCiMaterialSource", bulkPatchRequest)
		return
	}
	token := r.Header.Get("token")
	// Here passing the checkAppSpecificAccess func to check RBAC
	bulkPatchResponse, err := handler.pipelineBuilder.BulkPatchCiMaterialSource(bulkPatchRequest, userId, token, handler.checkAppSpecificAccess)
	if err != nil {
		handler.Logger.Errorw("service err, BulkPatchCiPipelines", "BulkPatchCiPipelines", bulkPatchRequest, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, bulkPatchResponse, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) PatchCiPipelines(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var patchRequest bean.CiPatchRequest
	err = decoder.Decode(&patchRequest)
	patchRequest.UserId = userId
	if err != nil {
		handler.Logger.Errorw("request err, PatchCiPipelines", "err", err, "PatchCiPipelines", patchRequest)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, PatchCiPipelines", "PatchCiPipelines", patchRequest)
	err = handler.validator.Struct(patchRequest)
	if err != nil {
		handler.Logger.Errorw("validation err", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Debugw("update request ", "req", patchRequest)
	token := r.Header.Get("token")
	app, err := handler.pipelineBuilder.GetApp(patchRequest.AppId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	appWorkflowName := ""
	if patchRequest.AppWorkflowId != 0 {
		appWorkflow, err := handler.appWorkflowService.FindAppWorkflowById(patchRequest.AppWorkflowId, app.Id)
		if err != nil {
			handler.Logger.Errorw("error in getting app workflow", "err", err, "workflowId", patchRequest.AppWorkflowId, "appId", app.Id)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
		appWorkflowName = appWorkflow.Name
	}
	resourceName := handler.enforcerUtil.GetAppRBACName(app.AppName)
	workflowResourceName := handler.enforcerUtil.GetRbacObjectNameByAppAndWorkflow(app.AppName, appWorkflowName)

	cdPipelines, err := handler.getCdPipelinesForCIPatchRbac(&patchRequest)
	if err != nil && err != pg.ErrNoRows {
		handler.Logger.Errorw("error in finding ccd cdPipelines by ciPipelineId", "ciPipelineId", patchRequest.CiPipeline.Id, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	haveCiPatchAccess := handler.checkCiPatchAccess(token, resourceName, cdPipelines)
	if !haveCiPatchAccess {
		haveCiPatchAccess = handler.enforcer.Enforce(token, casbin.ResourceJobs, casbin.ActionCreate, resourceName) && handler.enforcer.Enforce(token, casbin.ResourceWorkflow, casbin.ActionCreate, workflowResourceName)
	}
	if !haveCiPatchAccess {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}

	ciConf, err := handler.pipelineBuilder.GetCiPipeline(patchRequest.AppId)

	var emptyDockerRegistry string
	if app.AppType == helper.Job && ciConf == nil {
		ciConfigRequest := bean.CiConfigRequest{}
		ciConfigRequest.DockerRegistry = emptyDockerRegistry
		ciConfigRequest.AppId = patchRequest.AppId
		ciConfigRequest.CiBuildConfig = &bean2.CiBuildConfigBean{}
		ciConfigRequest.CiBuildConfig.CiBuildType = bean2.SKIP_BUILD_TYPE
		ciConfigRequest.UserId = patchRequest.UserId
		if patchRequest.CiPipeline == nil || patchRequest.CiPipeline.CiMaterial == nil {
			handler.Logger.Errorw("Invalid patch ci-pipeline request", "request", patchRequest, "err", "invalid CiPipeline data")
			common.WriteJsonResp(w, fmt.Errorf("invalid CiPipeline data"), nil, http.StatusBadRequest)
			return
		}
		ciConfigRequest.CiBuildConfig.GitMaterialId = patchRequest.CiPipeline.CiMaterial[0].GitMaterialId
		ciConfigRequest.IsJob = true
		_, err = handler.pipelineBuilder.CreateCiPipeline(&ciConfigRequest)
		if err != nil {
			handler.Logger.Errorw("error occurred in creating ci-pipeline for the Job", "payload", ciConfigRequest, "err", err)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
	}
	if app.AppType == helper.Job {
		patchRequest.IsJob = true
	}
	createResp, err := handler.pipelineBuilder.PatchCiPipeline(&patchRequest)
	if err != nil {
		if err.Error() == bean2.PIPELINE_NAME_ALREADY_EXISTS_ERROR {
			handler.Logger.Errorw("service err, pipeline name already exist ", "err", err)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
		handler.Logger.Errorw("service err, PatchCiPipelines", "PatchCiPipelines", patchRequest, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	if createResp != nil && app != nil {
		createResp.AppName = app.AppName
	}
	common.WriteJsonResp(w, err, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) getCdPipelinesForCIPatchRbac(patchRequest *bean.CiPatchRequest) (cdPipelines []*pipelineConfig.Pipeline, err error) {
	// if the request is for create, then there will be no cd pipelines created yet
	if patchRequest.IsCreateRequest() {
		return
	}
	// request is to either patch the existing pipeline or patch it by switching the source pipeline or delete or update-source

	// for switch , this API handles following switches
	// any -> any (except switching to external-ci)

	// to find the cd pipelines of the current ci pipelines workflow , we should query from appWorkflow Mappings.
	// cannot directly query cd-pipeline table as we don't store info about external pipeline in cdPipeline.

	// approach:
	// find the workflow in which we are patching and use the workflow id to fetch all the workflow mappings using the workflow.
	// get cd pipeline ids from those workflows and fetch the cd pipelines.

	// get the ciPipeline patch source info
	componentId, componentType := patchRequest.PatchSourceInfo()

	// the appWorkflowId can be taken from patchRequest.AppWorkflowId but doing this can make 2 sources of truth to find the workflow
	sourceAppWorkflowMapping, err := handler.appWorkflowService.FindWFMappingByComponent(componentType, componentId)
	if err != nil {
		handler.Logger.Errorw("error in finding the appWorkflowMapping using componentId and componentType", "componentType", componentType, "componentId", componentId, "err", err)
		return nil, err
	}

	cdPipelineWFMappings, err := handler.appWorkflowService.FindWFCDMappingsByWorkflowId(sourceAppWorkflowMapping.AppWorkflowId)
	if err != nil {
		handler.Logger.Errorw("error in finding the appWorkflowMappings of cd pipeline for an appWorkflow", "appWorkflowId", sourceAppWorkflowMapping.AppWorkflowId, "err", err)
		return cdPipelines, err
	}

	if len(cdPipelineWFMappings) == 0 {
		return
	}

	cdPipelineIds := make([]int, 0, len(cdPipelineWFMappings))
	for _, cdWfMapping := range cdPipelineWFMappings {
		cdPipelineIds = append(cdPipelineIds, cdWfMapping.ComponentId)
	}

	return handler.pipelineRepository.FindByIdsIn(cdPipelineIds)
}

// checkCiPatchAccess assumes all the cdPipelines belong to same app
func (handler *PipelineConfigRestHandlerImpl) checkCiPatchAccess(token string, resourceName string, cdPipelines []*pipelineConfig.Pipeline) bool {

	if len(cdPipelines) == 0 {
		// no cd pipelines are present , so user can edit if he has app admin access
		return handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionCreate, resourceName)
	}

	appId := 0
	envIds := make([]int, len(cdPipelines))
	for _, cdPipeline := range cdPipelines {
		envIds = append(envIds, cdPipeline.EnvironmentId)
		appId = cdPipeline.AppId
	}

	rbacObjectsMap, _ := handler.enforcerUtil.GetRbacObjectsByEnvIdsAndAppId(envIds, appId)
	envRbacResultMap := handler.enforcer.EnforceInBatch(token, casbin.ResourceEnvironment, casbin.ActionUpdate, maps.Values(rbacObjectsMap))

	for _, hasAccess := range envRbacResultMap {
		if hasAccess {
			return true
		}
	}

	return false
}

func (handler *PipelineConfigRestHandlerImpl) GetCiPipeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId, ok := handler.getIntPathParam(w, vars, "appId")
	if !ok {
		return
	}

	token := r.Header.Get("token")
	if !handler.checkAppRbacForAppOrJob(w, token, appId, casbin.ActionGet) {
		return
	}

	ciConf, err := handler.pipelineBuilder.GetCiPipelineRespResolved(appId)
	if err != nil {
		handler.Logger.Errorw("service err, GetCiPipelineRespResolved", "appId", appId, "err", err)
		common.WriteJsonResp(w, err, ciConf, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, ciConf, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetExternalCi(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId, ok := handler.getIntPathParam(w, vars, "appId")
	if !ok {
		return
	}

	token := r.Header.Get("token")
	_, authorized := handler.getAppAndCheckAuthForAction(w, appId, token, casbin.ActionGet)
	if !authorized {
		return
	}

	ciConf, err := handler.pipelineBuilder.GetExternalCi(appId)
	if err != nil {
		handler.Logger.Errorw("service err, GetExternalCi", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, ciConf, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetExternalCiById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId, ok := handler.getIntPathParam(w, vars, "appId")
	if !ok {
		return
	}

	externalCiId, ok := handler.getIntPathParam(w, vars, "externalCiId")
	if !ok {
		return
	}

	token := r.Header.Get("token")
	_, authorized := handler.getAppAndCheckAuthForAction(w, appId, token, casbin.ActionGet)
	if !authorized {
		return
	}

	ciConf, err := handler.pipelineBuilder.GetExternalCiById(appId, externalCiId)
	if err != nil {
		handler.Logger.Errorw("service err, GetExternalCiById", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, ciConf, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) validateCiTriggerRBAC(token string, ciPipelineId, triggerEnvironmentId int) error {
	// RBAC STARTS
	// checking if user has trigger access on app, if not will be forbidden to trigger independent of number of cd cdPipelines
	ciPipeline, err := handler.ciPipelineRepository.FindById(ciPipelineId)
	if err != nil {
		handler.Logger.Errorw("err in finding ci pipeline, TriggerCiPipeline", "err", err, "ciPipelineId", ciPipelineId)
		errMsg := fmt.Sprintf("error in finding ci pipeline for id '%d'", ciPipelineId)
		return util.NewApiError(http.StatusBadRequest, errMsg, errMsg)
	}
	appWorkflowMapping, err := handler.appWorkflowService.FindAppWorkflowByCiPipelineId(ciPipelineId)
	if err != nil {
		handler.Logger.Errorw("err in finding appWorkflowMapping, TriggerCiPipeline", "err", err, "ciPipelineId", ciPipelineId)
		errMsg := fmt.Sprintf("error in finding appWorkflowMapping for ciPipelineId '%d'", ciPipelineId)
		return util.NewApiError(http.StatusBadRequest, errMsg, errMsg)
	}
	workflowName := ""
	if len(appWorkflowMapping) > 0 {
		workflowName = appWorkflowMapping[0].AppWorkflow.Name
	}
	// This is being done for jobs, jobs execute in default-env (devtron-ci) namespace by default. so considering DefaultCiNamespace as env for rbac enforcement
	envName := ""
	if triggerEnvironmentId == 0 {
		envName = constants2.DefaultCiWorkflowNamespace
	}
	appObject := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	workflowObject := handler.enforcerUtil.GetWorkflowRBACByCiPipelineId(ciPipelineId, workflowName)
	triggerObject := handler.enforcerUtil.GetTeamEnvRBACNameByCiPipelineIdAndEnvIdOrName(ciPipelineId, triggerEnvironmentId, envName)
	var appRbacOk bool
	if ciPipeline.App.AppType == helper.CustomApp {
		appRbacOk = handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionTrigger, appObject)
	} else if ciPipeline.App.AppType == helper.Job {
		appRbacOk = handler.enforcer.Enforce(token, casbin.ResourceJobs, casbin.ActionTrigger, appObject) && handler.enforcer.Enforce(token, casbin.ResourceWorkflow, casbin.ActionTrigger, workflowObject) && handler.enforcer.Enforce(token, casbin.ResourceJobsEnv, casbin.ActionTrigger, triggerObject)
	}

	if !appRbacOk {
		handler.Logger.Debug(fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return util.NewApiError(http.StatusForbidden, common.UnAuthorisedUser, common.UnAuthorisedUser)
	}
	// checking rbac for cd cdPipelines
	cdPipelines, err := handler.pipelineRepository.FindByCiPipelineId(ciPipelineId)
	if err != nil {
		handler.Logger.Errorw("error in finding ccd cdPipelines by ciPipelineId", "err", err, "ciPipelineId", ciPipelineId)
		errMsg := fmt.Sprintf("error in finding cd cdPipelines for ciPipelineId '%d'", ciPipelineId)
		return util.NewApiError(http.StatusBadRequest, errMsg, errMsg)
	}
	cdPipelineRbacObjects := make([]string, len(cdPipelines))
	rbacObjectCdTriggerTypeMap := make(map[string]pipelineConfig.TriggerType, len(cdPipelines))
	for i, cdPipeline := range cdPipelines {
		envObject := handler.enforcerUtil.GetAppRBACByAppIdAndPipelineId(cdPipeline.AppId, cdPipeline.Id)
		cdPipelineRbacObjects[i] = envObject
		rbacObjectCdTriggerTypeMap[envObject] = cdPipeline.TriggerType
	}

	hasAnyEnvTriggerAccess := len(cdPipelines) == 0 //if no pipelines then appAccess is enough. For jobs also, this will be true
	if !hasAnyEnvTriggerAccess {
		//cdPipelines present, to check access for cd trigger
		envRbacResultMap := handler.enforcer.EnforceInBatch(token, casbin.ResourceEnvironment, casbin.ActionTrigger, cdPipelineRbacObjects)
		for rbacObject, rbacResultOk := range envRbacResultMap {
			if rbacObjectCdTriggerTypeMap[rbacObject] == pipelineConfig.TRIGGER_TYPE_AUTOMATIC && !rbacResultOk {
				return util.NewApiError(http.StatusForbidden, common.UnAuthorisedUser, common.UnAuthorisedUser)
			}
			if rbacResultOk { //this flow will come if pipeline is automatic and has access or if pipeline is manual,
				// by which we can ensure if there are no automatic pipelines then atleast access on one manual is present
				hasAnyEnvTriggerAccess = true
			}
		}
		if !hasAnyEnvTriggerAccess {
			return util.NewApiError(http.StatusForbidden, common.UnAuthorisedUser, common.UnAuthorisedUser)
		}
	}
	// RBAC ENDS
	return nil
}

func (handler *PipelineConfigRestHandlerImpl) TriggerCiPipeline(w http.ResponseWriter, r *http.Request) {
	userId, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	var ciTriggerRequest bean.CiTriggerRequest
	if !handler.decodeJsonBody(w, r, &ciTriggerRequest, "TriggerCiPipeline") {
		return
	}

	token := r.Header.Get("token")
	// RBAC block starts
	err := handler.validateCiTriggerRBAC(token, ciTriggerRequest.PipelineId, ciTriggerRequest.EnvironmentId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	// RBAC block ends

	if !handler.validForMultiMaterial(ciTriggerRequest) {
		handler.Logger.Errorw("invalid req, commit hash not present for multi-git", "payload", ciTriggerRequest)
		common.WriteJsonResp(w, errors.New("invalid req, commit hash not present for multi-git"),
			nil, http.StatusBadRequest)
		return
	}

	ciTriggerRequest.TriggeredBy = userId
	handler.Logger.Infow("request payload, TriggerCiPipeline", "payload", ciTriggerRequest)

	response := make(map[string]string)
	resp, err := handler.ciHandlerService.HandleCIManual(ciTriggerRequest)
	if errors.Is(err, bean1.ErrImagePathInUse) {
		handler.Logger.Errorw("service err duplicate image tag, TriggerCiPipeline", "err", err, "payload", ciTriggerRequest)
		common.WriteJsonResp(w, err, err, http.StatusConflict)
		return
	}

	if err != nil {
		handler.Logger.Errorw("service err, TriggerCiPipeline", "err", err, "payload", ciTriggerRequest)
		common.WriteJsonResp(w, err, response, http.StatusInternalServerError)
		return
	}

	response["apiResponse"] = strconv.Itoa(resp)
	common.WriteJsonResp(w, err, response, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FetchMaterials(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	pipelineId, ok := handler.getIntPathParam(w, vars, "pipelineId")
	if !ok {
		return
	}

	// Get showAll query parameter
	showAll := handler.getQueryParamBool(r, "showAll", false)

	handler.Logger.Infow("request payload, FetchMaterials", "pipelineId", pipelineId)

	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Errorw("service err, FindById", "err", err, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC check
	token := r.Header.Get("token")
	if !handler.checkAppRbacForAppOrJob(w, token, ciPipeline.AppId, casbin.ActionGet) {
		return
	}

	resp, err := handler.ciHandler.FetchMaterialsByPipelineId(pipelineId, showAll)
	if err != nil {
		handler.Logger.Errorw("service err", "err", err, "context", "FetchMaterials", "data", map[string]interface{}{"pipelineId": pipelineId})
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FetchMaterialsByMaterialId(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	pipelineId, ok := handler.getIntPathParam(w, vars, "pipelineId")
	if !ok {
		return
	}

	gitMaterialId, ok := handler.getIntPathParam(w, vars, "gitMaterialId")
	if !ok {
		return
	}

	// Get showAll query parameter
	showAll := handler.getQueryParamBool(r, "showAll", false)

	handler.Logger.Infow("request payload, FetchMaterialsByMaterialId", "pipelineId", pipelineId, "gitMaterialId", gitMaterialId)

	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Errorw("service err, FindById", "err", err, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC check
	token := r.Header.Get("token")
	if !handler.checkAppRbacForAppOrJob(w, token, ciPipeline.AppId, casbin.ActionGet) {
		return
	}

	resp, err := handler.ciHandler.FetchMaterialsByPipelineIdAndGitMaterialId(pipelineId, gitMaterialId, showAll)
	if err != nil {
		handler.Logger.Errorw("service err, FetchMaterials", "err", err, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) RefreshMaterials(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	gitMaterialId, ok := handler.getIntPathParam(w, vars, "gitMaterialId")
	if !ok {
		return
	}

	handler.Logger.Infow("request payload, RefreshMaterials", "gitMaterialId", gitMaterialId)

	material, err := handler.gitMaterialReadService.FindById(gitMaterialId)
	if err != nil {
		handler.Logger.Errorw("service err, RefreshMaterials", "err", err, "gitMaterialId", gitMaterialId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC check
	token := r.Header.Get("token")
	if !handler.checkAppRbacForAppOrJob(w, token, material.AppId, casbin.ActionGet) {
		return
	}

	resp, err := handler.ciHandler.RefreshMaterialByCiPipelineMaterialId(material.Id)
	if err != nil {
		handler.Logger.Errorw("service err, RefreshMaterials", "err", err, "gitMaterialId", gitMaterialId)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetCiPipelineMin(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	appId, ok := handler.getIntPathParam(w, vars, "appId")
	if !ok {
		return
	}

	// Parse environment IDs from query parameter
	v := r.URL.Query()
	envIdsString := v.Get("envIds")
	envIds := make([]int, 0)
	if len(envIdsString) > 0 {
		var err error
		envIds, err = stringsUtil.SplitCommaSeparatedIntValues(envIdsString)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid envIds", http.StatusBadRequest)
			return
		}
	}

	handler.Logger.Infow("request payload, GetCiPipelineMin", "appId", appId)

	// RBAC check
	token := r.Header.Get("token")
	if !handler.checkAppRbacForAppOrJob(w, token, appId, casbin.ActionGet) {
		return
	}

	ciPipelines, err := handler.pipelineBuilder.GetCiPipelineMin(appId, envIds)
	if err != nil {
		handler.Logger.Errorw("service err, GetCiPipelineMin", "err", err, "appId", appId)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: http.StatusNotFound, UserMessage: "no data found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
			return
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
	}
	common.WriteJsonResp(w, nil, ciPipelines, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) DownloadCiWorkflowArtifacts(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	pipelineId, ok := handler.getIntPathParam(w, vars, "pipelineId")
	if !ok {
		return
	}

	buildId, ok := handler.getIntPathParam(w, vars, "workflowId")
	if !ok {
		return
	}

	handler.Logger.Infow("request payload, DownloadCiWorkflowArtifacts", "pipelineId", pipelineId, "buildId", buildId)

	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Error(err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC check
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, nil, "Unauthorized User", http.StatusForbidden)
		return
	}

	file, err := handler.ciHandlerService.DownloadCiWorkflowArtifacts(pipelineId, buildId)
	defer file.Close()
	if err != nil {
		handler.Logger.Errorw("service err, DownloadCiWorkflowArtifacts", "err", err, "pipelineId", pipelineId, "buildId", buildId)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no workflow found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Itoa(buildId)+".zip")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", r.Header.Get("Content-Length"))

	_, err = io.Copy(w, file)
	if err != nil {
		handler.Logger.Errorw("service err, DownloadCiWorkflowArtifacts", "err", err, "pipelineId", pipelineId, "buildId", buildId)
	}
}

func (handler *PipelineConfigRestHandlerImpl) GetHistoricBuildLogs(w http.ResponseWriter, r *http.Request) {
	_, ok := handler.getUserIdOrUnauthorized(w, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	pipelineId, ok := handler.getIntPathParam(w, vars, "pipelineId")
	if !ok {
		return
	}

	workflowId, ok := handler.getIntPathParam(w, vars, "workflowId")
	if !ok {
		return
	}

	handler.Logger.Infow("request payload, GetHistoricBuildLogs", "pipelineId", pipelineId, "workflowId", workflowId)

	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Errorw("service err, GetHistoricBuildLogs", "err", err, "pipelineId", pipelineId, "workflowId", workflowId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	// RBAC check
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, nil, "Unauthorized User", http.StatusForbidden)
		return
	}
	// RBAC
	resp, err := handler.ciHandlerService.GetHistoricBuildLogs(workflowId, nil)
	if err != nil {
		handler.Logger.Errorw("service err, GetHistoricBuildLogs", "err", err, "pipelineId", pipelineId, "workflowId", workflowId)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetBuildHistory(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	offsetQueryParam := r.URL.Query().Get("offset")
	offset, err := strconv.Atoi(offsetQueryParam)
	if offsetQueryParam == "" || err != nil {
		common.WriteJsonResp(w, err, "invalid offset", http.StatusBadRequest)
		return
	}
	sizeQueryParam := r.URL.Query().Get("size")
	limit, err := strconv.Atoi(sizeQueryParam)
	if sizeQueryParam == "" || err != nil {
		common.WriteJsonResp(w, err, "invalid size", http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, GetBuildHistory", "pipelineId", pipelineId, "offset", offset)
	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Errorw("service err, GetBuildHistory", "err", err, "pipelineId", pipelineId, "offset", offset)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	appWorkflowMapping, err := handler.appWorkflowService.FindAppWorkflowByCiPipelineId(pipelineId)
	if err != nil {
		handler.Logger.Errorw("service err, GetBuildHistory", "err", err, "pipelineId", pipelineId, "offset", offset)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//RBAC for build history
	token := r.Header.Get("token")
	isAuthorised := false
	workflowName := ""
	if len(appWorkflowMapping) > 0 {
		workflowName = appWorkflowMapping[0].AppWorkflow.Name
	}
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	workflowResourceObject := handler.enforcerUtil.GetWorkflowRBACByCiPipelineId(pipelineId, workflowName)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); ok {
		isAuthorised = true
	}
	if !isAuthorised {
		isAuthorised = handler.enforcer.Enforce(token, casbin.ResourceJobs, casbin.ActionGet, object) && handler.enforcer.Enforce(token, casbin.ResourceWorkflow, casbin.ActionGet, workflowResourceObject)
	}
	if !isAuthorised {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC
	//RBAC for edit tag access , user should have build permission in current ci-pipeline
	triggerAccess := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionTrigger, object) || handler.enforcer.Enforce(token, casbin.ResourceJobs, casbin.ActionTrigger, object)
	//RBAC
	resp := apiBean.BuildHistoryResponse{}
	workflowsResp, err := handler.ciHandler.GetBuildHistory(pipelineId, ciPipeline.AppId, offset, limit)
	resp.CiWorkflows = workflowsResp
	if err != nil {
		handler.Logger.Errorw("service err, GetBuildHistory", "err", err, "pipelineId", pipelineId, "offset", offset)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	appTags, err := handler.imageTaggingReadService.GetUniqueTagsByAppId(ciPipeline.AppId)
	if err != nil {
		handler.Logger.Errorw("service err, GetTagsByAppId", "err", err, "appId", ciPipeline.AppId)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	resp.AppReleaseTagNames = appTags

	prodEnvExists, err := handler.imageTaggingService.GetProdEnvFromParentAndLinkedWorkflow(ciPipeline.Id)
	resp.TagsEditable = prodEnvExists && triggerAccess
	resp.HideImageTaggingHardDelete = handler.imageTaggingService.IsHardDeleteHidden()
	if err != nil {
		handler.Logger.Errorw("service err, GetProdEnvFromParentAndLinkedWorkflow", "err", err, "ciPipelineId", ciPipeline.Id)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetBuildLogs(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	followLogs := true
	if ok := r.URL.Query().Has("followLogs"); ok {
		followLogsStr := r.URL.Query().Get("followLogs")
		follow, err := strconv.ParseBool(followLogsStr)
		if err != nil {
			common.WriteJsonResp(w, err, "followLogs is not a valid bool", http.StatusBadRequest)
			return
		}
		followLogs = follow
	}

	workflowId, err := strconv.Atoi(vars["workflowId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, GetBuildLogs", "pipelineId", pipelineId, "workflowId", workflowId)
	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC
	lastSeenMsgId := -1
	lastEventId := r.Header.Get("Last-Event-ID")
	if len(lastEventId) > 0 {
		lastSeenMsgId, err = strconv.Atoi(lastEventId)
		if err != nil {
			handler.Logger.Errorw("request err, GetBuildLogs", "err", err, "pipelineId", pipelineId, "workflowId", workflowId, "lastEventId", lastEventId)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}
	logsReader, cleanUp, err := handler.ciHandlerService.GetRunningWorkflowLogs(workflowId, followLogs)
	if err != nil {
		handler.Logger.Errorw("service err, GetBuildLogs", "err", err, "pipelineId", pipelineId, "workflowId", workflowId, "lastEventId", lastEventId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	if cn, ok := w.(http.CloseNotifier); ok {
		go func(done <-chan struct{}, closed <-chan bool) {
			select {
			case <-done:
			case <-closed:
				cancel()
			}
		}(ctx.Done(), cn.CloseNotify())
	}
	defer cancel()
	defer func() {
		if cleanUp != nil {
			cleanUp()
		}
	}()
	handler.streamOutput(w, logsReader, lastSeenMsgId)
}

func (handler *PipelineConfigRestHandlerImpl) FetchMaterialInfo(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	ciArtifactId, err := strconv.Atoi(vars["ciArtifactId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, FetchMaterialInfo", "err", err, "ciArtifactId", ciArtifactId)
	resp, err := handler.ciHandler.FetchMaterialInfoByArtifactId(ciArtifactId, envId)
	if err != nil {
		handler.Logger.Errorw("service err, FetchMaterialInfo", "err", err, "ciArtifactId", ciArtifactId)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: http.StatusNotFound, UserMessage: "no material info found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(resp.AppId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC

	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetCIPipelineById(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	handler.Logger.Infow("request payload, GetCIPipelineById", "err", err, "appId", appId, "pipelineId", pipelineId)

	app, err := handler.pipelineBuilder.GetApp(appId)
	if err != nil {
		handler.Logger.Infow("service error, GetCIPipelineById", "err", err, "appId", appId, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	resourceName := handler.enforcerUtil.GetAppRBACName(app.AppName)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceName, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}

	pipelineData, err := handler.pipelineRepository.FindActiveByAppIdAndPipelineId(appId, pipelineId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	var environmentIds []int
	for _, pipeline := range pipelineData {
		environmentIds = append(environmentIds, pipeline.EnvironmentId)
	}
	if handler.appWorkflowService.CheckCdPipelineByCiPipelineId(pipelineId) {
		for _, envId := range environmentIds {
			envObject := handler.enforcerUtil.GetEnvRBACNameByCiPipelineIdAndEnvId(pipelineId, envId)
			if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionGet, envObject); !ok {
				common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
				return
			}
		}
	}

	ciPipeline, err := handler.pipelineBuilder.GetCiPipelineByIdWithDefaultTag(pipelineId)
	if err != nil {
		handler.Logger.Infow("service error, GetCiPipelineByIdWithDefaultTag", "err", err, "appId", appId, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, ciPipeline, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetCIPipelineByPipelineId(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	var ciPipelineId int
	var err error
	v := r.URL.Query()
	pipelineId := v.Get("pipelineId")
	if len(pipelineId) != 0 {
		ciPipelineId, err = strconv.Atoi(pipelineId)
		if err != nil {
			handler.Logger.Errorw("request err, GetCIPipelineByPipelineId", "err", err, "pipelineIdParam", pipelineId)
			response.WriteResponse(http.StatusBadRequest, "please send valid pipelineId", w, errors.New("pipelineId id invalid"))
			return
		}
	} else {
		response.WriteResponse(http.StatusBadRequest, "please send valid pipelineId", w, errors.New("pipelineId id invalid"))
		return
	}

	handler.Logger.Infow("request payload, GetCIPipelineByPipelineId", "pipelineId", pipelineId)

	ciPipeline, err := handler.pipelineBuilder.GetCiPipelineById(ciPipelineId)
	if err != nil {
		handler.Logger.Infow("service error, GetCIPipelineById", "err", err, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	app, err := handler.pipelineBuilder.GetApp(ciPipeline.AppId)
	if err != nil {
		handler.Logger.Infow("service error, GetCIPipelineByPipelineId", "err", err, "appId", ciPipeline.AppId, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	ciPipeline.AppName = app.AppName
	ciPipeline.AppType = app.AppType

	resourceName := handler.enforcerUtil.GetAppRBACNameByAppId(app.Id)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, resourceName); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}

	pipelineData, err := handler.pipelineRepository.FindActiveByAppIdAndPipelineId(ciPipeline.AppId, ciPipelineId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	var environmentIds []int
	for _, pipeline := range pipelineData {
		environmentIds = append(environmentIds, pipeline.EnvironmentId)
	}
	if handler.appWorkflowService.CheckCdPipelineByCiPipelineId(ciPipelineId) {
		for _, envId := range environmentIds {
			envObject := handler.enforcerUtil.GetEnvRBACNameByCiPipelineIdAndEnvId(ciPipelineId, envId)
			if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionUpdate, envObject); !ok {
				common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
				return
			}
		}
	}
	common.WriteJsonResp(w, err, ciPipeline, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) CreateMaterial(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var createMaterialDto bean.CreateMaterialDTO
	err = decoder.Decode(&createMaterialDto)
	createMaterialDto.UserId = userId
	if err != nil {
		handler.Logger.Errorw("request err, CreateMaterial", "err", err, "CreateMaterial", createMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, CreateMaterial", "CreateMaterial", createMaterialDto)
	err = handler.validator.Struct(createMaterialDto)
	if err != nil {
		handler.Logger.Errorw("validation err, CreateMaterial", "err", err, "CreateMaterial", createMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	resourceObject := handler.enforcerUtil.GetAppRBACNameByAppId(createMaterialDto.AppId)
	isAuthorised := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceObject, casbin.ActionCreate)
	if !isAuthorised {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	for _, gitMaterial := range createMaterialDto.Material {
		validationResult, err := handler.ValidateGitMaterialUrl(gitMaterial.GitProviderId, gitMaterial.Url)
		if err != nil {
			handler.Logger.Errorw("service err, CreateMaterial", "err", err, "CreateMaterial", createMaterialDto)
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		} else {
			if !validationResult {
				handler.Logger.Errorw("validation err, CreateMaterial : invalid git material url", "err", err, "gitMaterialUrl", gitMaterial.Url, "CreateMaterial", createMaterialDto)
				common.WriteJsonResp(w, fmt.Errorf("validation for url failed"), nil, http.StatusBadRequest)
				return
			}
		}
	}

	createResp, err := handler.pipelineBuilder.CreateMaterialsForApp(&createMaterialDto)
	if err != nil {
		handler.Logger.Errorw("service err, CreateMaterial", "err", err, "CreateMaterial", createMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) UpdateMaterial(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var updateMaterialDto bean.UpdateMaterialDTO
	err = decoder.Decode(&updateMaterialDto)
	updateMaterialDto.UserId = userId
	if err != nil {
		handler.Logger.Errorw("request err, UpdateMaterial", "err", err, "UpdateMaterial", updateMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, UpdateMaterial", "UpdateMaterial", updateMaterialDto)
	err = handler.validator.Struct(updateMaterialDto)
	if err != nil {
		handler.Logger.Errorw("validation err, UpdateMaterial", "err", err, "UpdateMaterial", updateMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	validationResult, err := handler.ValidateGitMaterialUrl(updateMaterialDto.Material.GitProviderId, updateMaterialDto.Material.Url)
	if err != nil {
		handler.Logger.Errorw("service err, UpdateMaterial", "err", err, "UpdateMaterial", updateMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	} else {
		if !validationResult {
			handler.Logger.Errorw("validation err, UpdateMaterial : invalid git material url", "err", err, "gitMaterialUrl", updateMaterialDto.Material.Url, "UpdateMaterial", updateMaterialDto)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}
	resourceObject := handler.enforcerUtil.GetAppRBACNameByAppId(updateMaterialDto.AppId)
	isAuthorised := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceObject, casbin.ActionCreate)
	if !isAuthorised {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}

	createResp, err := handler.pipelineBuilder.UpdateMaterialsForApp(&updateMaterialDto)
	if err != nil {
		handler.Logger.Errorw("service err, UpdateMaterial", "err", err, "UpdateMaterial", updateMaterialDto)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, createResp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) DeleteMaterial(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var deleteMaterial bean.UpdateMaterialDTO
	err = decoder.Decode(&deleteMaterial)
	deleteMaterial.UserId = userId
	if err != nil {
		handler.Logger.Errorw("request err, DeleteMaterial", "err", err, "DeleteMaterial", deleteMaterial)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, DeleteMaterial", "DeleteMaterial", deleteMaterial)
	err = handler.validator.Struct(deleteMaterial)
	if err != nil {
		handler.Logger.Errorw("validation err, DeleteMaterial", "err", err, "DeleteMaterial", deleteMaterial)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	//rbac starts
	resourceObject := handler.enforcerUtil.GetAppRBACNameByAppId(deleteMaterial.AppId)
	token := r.Header.Get("token")
	if ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, resourceObject, casbin.ActionCreate); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//rbac ends
	err = handler.pipelineBuilder.DeleteMaterial(&deleteMaterial)
	if err != nil {
		handler.Logger.Errorw("service err, DeleteMaterial", "err", err, "DeleteMaterial", deleteMaterial)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, apiBean.GIT_MATERIAL_DELETE_SUCCESS_RESP, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) HandleWorkflowWebhook(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var wfUpdateReq eventProcessorBean.CiCdStatus
	err := decoder.Decode(&wfUpdateReq)
	if err != nil {
		handler.Logger.Errorw("request err, HandleWorkflowWebhook", "err", err, "payload", wfUpdateReq)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, HandleWorkflowWebhook", "payload", wfUpdateReq)
	resp, _, err := handler.ciHandler.UpdateWorkflow(wfUpdateReq)
	if err != nil {
		handler.Logger.Errorw("service err, HandleWorkflowWebhook", "err", err, "payload", wfUpdateReq)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) validForMultiMaterial(ciTriggerRequest bean.CiTriggerRequest) bool {
	if len(ciTriggerRequest.CiPipelineMaterial) > 1 {
		for _, m := range ciTriggerRequest.CiPipelineMaterial {
			if m.GitCommit.Commit == "" {
				return false
			}
		}
	}
	return true
}

func (handler *PipelineConfigRestHandlerImpl) ValidateGitMaterialUrl(gitProviderId int, url string) (bool, error) {
	gitProvider, err := handler.gitProviderReadService.FetchOneGitProvider(strconv.Itoa(gitProviderId))
	if err != nil {
		return false, err
	}
	if gitProvider.AuthMode == constants.AUTH_MODE_SSH {
		// this regex is used to generic ssh providers like gogs where format is <user>@<host>:<org>/<repo>.git
		var scpLikeSSHRegex = regexp.MustCompile(`^[\w-]+@[\w.-]+:[\w./-]+\.git$`)
		hasPrefixResult := strings.HasPrefix(url, SSH_URL_PREFIX) || scpLikeSSHRegex.MatchString(url)
		return hasPrefixResult, nil
	}
	hasPrefixResult := strings.HasPrefix(url, HTTPS_URL_PREFIX) || strings.HasPrefix(url, HTTP_URL_PREFIX)
	return hasPrefixResult, nil
}

func (handler *PipelineConfigRestHandlerImpl) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	queryVars := r.URL.Query()
	vars := mux.Vars(r)
	workflowId, err := strconv.Atoi(vars["workflowId"])
	if err != nil {
		handler.Logger.Errorw("request err, CancelWorkflow", "err", err, "workflowId", workflowId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		handler.Logger.Errorw("request err, CancelWorkflow", "err", err, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	var forceAbort bool
	forceAbortQueryParam := queryVars.Get("forceAbort")
	if len(forceAbortQueryParam) > 0 {
		forceAbort, err = strconv.ParseBool(forceAbortQueryParam)
		if err != nil {
			handler.Logger.Errorw("request err, CancelWorkflow", "err", err)
			common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
			return
		}
	}

	handler.Logger.Infow("request payload, CancelWorkflow", "workflowId", workflowId, "pipelineId", pipelineId)

	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Errorw("service err, CancelWorkflow", "err", err, "workflowId", workflowId, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionTrigger)
	if !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	if handler.appWorkflowService.CheckCdPipelineByCiPipelineId(pipelineId) {
		pipelineData, err := handler.pipelineRepository.FindActiveByAppIdAndPipelineId(ciPipeline.AppId, pipelineId)
		if err != nil {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
			return
		}
		var environmentIds []int
		for _, pipeline := range pipelineData {
			environmentIds = append(environmentIds, pipeline.EnvironmentId)
		}
		if handler.appWorkflowService.CheckCdPipelineByCiPipelineId(pipelineId) {
			for _, envId := range environmentIds {
				envObject := handler.enforcerUtil.GetEnvRBACNameByCiPipelineIdAndEnvId(pipelineId, envId)
				if ok := handler.enforcer.Enforce(token, casbin.ResourceEnvironment, casbin.ActionUpdate, envObject); !ok {
					common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
					return
				}
			}
		}
	}

	//RBAC

	resp, err := handler.ciHandlerService.CancelBuild(workflowId, forceAbort)
	if err != nil {
		handler.Logger.Errorw("service err, CancelWorkflow", "err", err, "workflowId", workflowId, "pipelineId", pipelineId)
		if util.IsErrNoRows(err) {
			common.WriteJsonResp(w, err, nil, http.StatusNotFound)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

// FetchChanges FIXME check if deprecated
func (handler *PipelineConfigRestHandlerImpl) FetchChanges(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	ciMaterialId, err := strconv.Atoi(vars["ciMaterialId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	showAll := false
	v := r.URL.Query()
	show := v.Get("showAll")
	if len(show) > 0 {
		showAll, err = strconv.ParseBool(show)
		if err != nil {
			showAll = true
			err = nil
			//ignore error, apply rbac by default
		}
	}
	handler.Logger.Infow("request payload, FetchChanges", "ciMaterialId", ciMaterialId, "pipelineId", pipelineId)
	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		handler.Logger.Errorw("request err, FetchChanges", "err", err, "ciMaterialId", ciMaterialId, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC

	changeRequest := &gitSensor.FetchScmChangesRequest{
		PipelineMaterialId: ciMaterialId,
		ShowAll:            showAll,
	}
	changes, err := handler.gitSensorClient.FetchChanges(context.Background(), changeRequest)
	if err != nil {
		handler.Logger.Errorw("service err, FetchChanges", "err", err, "ciMaterialId", ciMaterialId, "pipelineId", pipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, changes.Commits, http.StatusCreated)
}

func (handler *PipelineConfigRestHandlerImpl) GetCommitMetadataForPipelineMaterial(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	ciPipelineMaterialId, err := strconv.Atoi(vars["ciPipelineMaterialId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	gitHash := vars["gitHash"]
	handler.Logger.Infow("request payload, GetCommitMetadataForPipelineMaterial", "ciPipelineMaterialId", ciPipelineMaterialId, "gitHash", gitHash)

	// get ci-pipeline-material
	ciPipelineMaterial, err := handler.ciPipelineMaterialRepository.GetById(ciPipelineMaterialId)
	if err != nil {
		handler.Logger.Errorw("error while fetching ciPipelineMaterial", "err", err, "ciPipelineMaterialId", ciPipelineMaterialId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipelineMaterial.CiPipeline.AppId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC

	commitMetadataRequest := &gitSensor.CommitMetadataRequest{
		PipelineMaterialId: ciPipelineMaterialId,
		GitHash:            gitHash,
	}
	commit, err := handler.gitSensorClient.GetCommitMetadataForPipelineMaterial(context.Background(), commitMetadataRequest)
	if err != nil {
		handler.Logger.Errorw("error while fetching commit metadata for pipeline material", "commitMetadataRequest", commitMetadataRequest, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, commit, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) FetchWorkflowDetails(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	buildId, err := strconv.Atoi(vars["workflowId"])
	if err != nil || buildId == 0 {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, FetchWorkflowDetails", "appId", appId, "pipelineId", pipelineId, "buildId", buildId, "buildId", buildId)
	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	ok := handler.enforcerUtil.CheckAppRbacForAppOrJob(token, object, casbin.ActionGet)
	if !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC
	resp, err := handler.ciHandler.FetchWorkflowDetails(appId, pipelineId, buildId)
	if err != nil {
		handler.Logger.Errorw("service err, FetchWorkflowDetails", "err", err, "appId", appId, "pipelineId", pipelineId, "buildId", buildId, "buildId", buildId)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: http.StatusNotFound, UserMessage: "no workflow found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetArtifactsForCiJob(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	pipelineId, err := strconv.Atoi(vars["pipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	buildId, err := strconv.Atoi(vars["workflowId"])
	if err != nil || buildId == 0 {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	handler.Logger.Infow("request payload, GetArtifactsForCiJob", "pipelineId", pipelineId, "buildId", buildId, "buildId", buildId)
	ciPipeline, err := handler.ciPipelineRepository.FindById(pipelineId)
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//RBAC
	token := r.Header.Get("token")
	object := handler.enforcerUtil.GetAppRBACNameByAppId(ciPipeline.AppId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, object); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC
	resp, err := handler.ciHandler.FetchArtifactsForCiJob(buildId)
	if err != nil {
		handler.Logger.Errorw("service err, FetchArtifactsForCiJob", "err", err, "pipelineId", pipelineId, "buildId", buildId, "buildId", buildId)
		if util.IsErrNoRows(err) {
			err = &util.ApiError{Code: "404", HttpStatusCode: http.StatusNotFound, UserMessage: "no artifact found"}
			common.WriteJsonResp(w, err, nil, http.StatusOK)
		} else {
			common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		}
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetCiPipelineByEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, GetCdPipelines", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}
	var appGroupId int
	appGroupIdStr := v.Get("appGroupId")
	if len(appGroupIdStr) > 0 {
		appGroupId, err = strconv.Atoi(appGroupIdStr)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid appGroupId", http.StatusBadRequest)
			return
		}
	}

	request := resourceGroup.ResourceGroupingRequest{
		ParentResourceId:  envId,
		ResourceGroupId:   appGroupId,
		ResourceGroupType: resourceGroup.APP_GROUP,
		ResourceIds:       appIds,
		CheckAuthBatch:    handler.checkAuthBatch,
		UserId:            userId,
		Ctx:               r.Context(),
	}
	_, span := otel.Tracer("orchestrator").Start(r.Context(), "ciHandler.FetchCiPipelinesForAppGrouping")
	ciConf, err := handler.pipelineBuilder.GetCiPipelineByEnvironment(request, token)
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err, GetCiPipeline", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, ciConf, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetCiPipelineByEnvironmentMin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, GetCdPipelines", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}
	var appGroupId int
	appGroupIdStr := v.Get("appGroupId")
	if len(appGroupIdStr) > 0 {
		appGroupId, err = strconv.Atoi(appGroupIdStr)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid appGroupId", http.StatusBadRequest)
			return
		}
	}
	request := resourceGroup.ResourceGroupingRequest{
		ParentResourceId:  envId,
		ResourceGroupId:   appGroupId,
		ResourceGroupType: resourceGroup.APP_GROUP,
		ResourceIds:       appIds,
		CheckAuthBatch:    handler.checkAuthBatch,
		UserId:            userId,
		Ctx:               r.Context(),
	}
	_, span := otel.Tracer("orchestrator").Start(r.Context(), "ciHandler.FetchCiPipelinesForAppGrouping")
	results, err := handler.pipelineBuilder.GetCiPipelineByEnvironmentMin(request, token)
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err, GetCiPipeline", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, results, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetExternalCiByEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}

	var appGroupId int
	appGroupIdStr := v.Get("appGroupId")
	if len(appGroupIdStr) > 0 {
		appGroupId, err = strconv.Atoi(appGroupIdStr)
		if err != nil {
			common.WriteJsonResp(w, err, "please provide valid appGroupId", http.StatusBadRequest)
			return
		}
	}
	request := resourceGroup.ResourceGroupingRequest{
		ParentResourceId:  envId,
		ResourceGroupId:   appGroupId,
		ResourceGroupType: resourceGroup.APP_GROUP,
		ResourceIds:       appIds,
		CheckAuthBatch:    handler.checkAuthBatch,
		UserId:            userId,
		Ctx:               r.Context(),
	}
	_, span := otel.Tracer("orchestrator").Start(r.Context(), "ciHandler.FetchExternalCiPipelinesForAppGrouping")
	ciConf, err := handler.pipelineBuilder.GetExternalCiByEnvironment(request, token)
	span.End()
	if err != nil {
		handler.Logger.Errorw("service err, GetExternalCi", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, ciConf, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) CreateUpdateImageTagging(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")

	artifactId, err := strconv.Atoi(vars["artifactId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	externalCi, ciPipelineId, appId, err := handler.extractCipipelineMetaForImageTags(artifactId)
	if err != nil {
		handler.Logger.Errorw("error occurred in fetching extractCipipelineMetaForImageTags by artifact Id ", "err", err, "artifactId", artifactId)
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusInternalServerError)
		return
	}

	decoder := json.NewDecoder(r.Body)
	req := &types.ImageTaggingRequestDTO{}
	err = decoder.Decode(&req)
	if err != nil {
		handler.Logger.Errorw("request err, CreateUpdateImageTagging", "err", err, "payload", req)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	//RBAC
	if !isSuperAdmin {
		object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
		if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionTrigger, object); !ok {
			common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
			return
		}
	}
	//RBAC
	//check prod env exists
	prodEnvExists := false
	if externalCi {
		prodEnvExists, err = handler.imageTaggingService.FindProdEnvExists(true, []int{ciPipelineId})
	} else {
		prodEnvExists, err = handler.imageTaggingService.GetProdEnvFromParentAndLinkedWorkflow(ciPipelineId)
	}
	if err != nil {
		handler.Logger.Errorw("error occurred in checking existence of prod environment ", "err", err, "ciPipelineId", ciPipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	//not allowed to perform edit/save if no cd exists in prod env in the app_workflow
	if !prodEnvExists {
		handler.Logger.Errorw("save or edit operation not possible for this artifact", "err", nil, "artifactId", artifactId, "ciPipelineId", ciPipelineId)
		common.WriteJsonResp(w, errors.New("save or edit operation not possible for this artifact"), nil, http.StatusBadRequest)
		return
	}

	if !isSuperAdmin && len(req.HardDeleteTags) > 0 {
		errMsg := errors.New("user dont have permission to delete the tags")
		handler.Logger.Errorw("request err, CreateUpdateImageTagging", "err", errMsg, "payload", req)
		common.WriteJsonResp(w, errMsg, nil, http.StatusBadRequest)
		return
	}
	//validate request
	isValidRequest, err := handler.imageTaggingService.ValidateImageTaggingRequest(req, appId, artifactId)
	if err != nil || !isValidRequest {
		handler.Logger.Errorw("request validation failed", "error", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	req.ExternalCi = externalCi
	// pass it to the service layer
	resp, err := handler.imageTaggingService.CreateOrUpdateImageTagging(ciPipelineId, appId, artifactId, int(userId), req)
	if err != nil {
		if err.Error() == imageTagging.DuplicateTagsInAppError {
			appReleaseTags, err1 := handler.imageTaggingReadService.GetUniqueTagsByAppId(appId)
			if err1 != nil {
				handler.Logger.Errorw("error occurred in getting unique tags in app", "err", err1, "appId", appId)
				err = err1
			}
			resp = &types.ImageTaggingResponseDTO{}
			resp.AppReleaseTags = appReleaseTags
		}
		handler.Logger.Errorw("error occurred in creating/updating image tagging data", "err", err, "ciPipelineId", ciPipelineId)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetImageTaggingData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	artifactId, err := strconv.Atoi(vars["artifactId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	pipelineId, err := strconv.Atoi(vars["ciPipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	externalCi, ciPipelineId, appId, err := handler.extractCipipelineMetaForImageTags(artifactId)
	if err != nil {
		handler.Logger.Errorw("error occurred in fetching extract ci pipeline metadata for ImageTags by artifact id", "err", err, "artifactId", artifactId)
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusInternalServerError)
		return
	}
	if !externalCi && (ciPipelineId != pipelineId) {
		common.WriteJsonResp(w, errors.New("ciPipelineId and artifactId sent in the request are not related"), nil, http.StatusBadRequest)
		return
	}
	//RBAC
	object := handler.enforcerUtil.GetAppRBACNameByAppId(appId)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionTrigger, object); !ok {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusForbidden)
		return
	}
	//RBAC

	resp, err := handler.imageTaggingService.GetTagsData(ciPipelineId, appId, artifactId, externalCi)
	if err != nil {
		handler.Logger.Errorw("error occurred in fetching GetTagsData for artifact ", "err", err, "artifactId", artifactId, "ciPipelineId", ciPipelineId, "externalCi", externalCi, "appId", appId)
		common.WriteJsonResp(w, err, resp, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) extractCipipelineMetaForImageTags(artifactId int) (externalCi bool, ciPipelineId int, appId int, err error) {
	externalCi = false
	ciPipelineId = 0
	appId = 0
	ciArtifact, err := handler.ciArtifactRepository.Get(artifactId)
	if err != nil {
		handler.Logger.Errorw("Error in fetching ci artifact by ci artifact id", "err", err)
		return externalCi, ciPipelineId, appId, err
	}
	if ciArtifact.DataSource == repository.POST_CI {
		ciPipelineId = ciArtifact.ComponentId
		ciPipeline, err := handler.pipelineBuilder.GetCiPipelineById(ciPipelineId)
		if err != nil {
			handler.Logger.Errorw("no ci pipeline found for given artifact", "err", err, "artifactId", artifactId, "ciPipelineId", ciPipelineId)
			return externalCi, ciPipelineId, appId, err
		}
		appId = ciPipeline.AppId
	} else if ciArtifact.DataSource == repository.PRE_CD || ciArtifact.DataSource == repository.POST_CD {
		cdPipelineId := ciArtifact.ComponentId
		cdPipeline, err := handler.pipelineBuilder.GetCdPipelineById(cdPipelineId)
		if err != nil {
			handler.Logger.Errorw("no cd pipeline found for given artifact", "err", err, "artifactId", artifactId, "cdPipelineId", cdPipelineId)
			return externalCi, ciPipelineId, appId, err
		}
		ciPipelineId = cdPipeline.CiPipelineId
		appId = cdPipeline.AppId
	} else {
		ciPipeline, err := handler.ciPipelineRepository.GetCiPipelineByArtifactId(artifactId)
		var externalCiPipeline *pipelineConfig.ExternalCiPipeline
		if err != nil {
			if err == pg.ErrNoRows {
				handler.Logger.Infow("no ciPipeline found by artifact Id, fetching external ci-pipeline ", "artifactId", artifactId)
				externalCiPipeline, err = handler.ciPipelineRepository.GetExternalCiPipelineByArtifactId(artifactId)
			}
			if err != nil {
				handler.Logger.Errorw("error occurred in fetching ciPipeline/externalCiPipeline by artifact Id ", "err", err, "artifactId", artifactId)
				return externalCi, ciPipelineId, appId, err
			}
		}
		if ciPipeline.Id != 0 {
			ciPipelineId = ciPipeline.Id
			appId = ciPipeline.AppId
		} else {
			externalCi = true
			ciPipelineId = externalCiPipeline.Id
			appId = externalCiPipeline.AppId
		}
	}
	return externalCi, ciPipelineId, appId, nil
}

func (handler *PipelineConfigRestHandlerImpl) checkAppSpecificAccess(token, action string, appId int) (bool, error) {
	app, err := handler.pipelineBuilder.GetApp(appId)
	if err != nil {
		return false, err
	}
	if app.AppType != helper.CustomApp {
		return false, errors.New("only custom apps supported")
	}

	resourceName := handler.enforcerUtil.GetAppRBACName(app.AppName)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, action, resourceName); !ok {
		return false, errors.New(string(bean.CI_PATCH_NOT_AUTHORIZED_MESSAGE))
	}
	return true, nil
}

func (handler *PipelineConfigRestHandlerImpl) GetSourceCiDownStreamFilters(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	ciPipelineId, err := strconv.Atoi(vars["ciPipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	ciPipeline, err := handler.ciPipelineRepository.FindOneWithAppData(ciPipelineId)
	if util.IsErrNoRows(err) {
		common.WriteJsonResp(w, fmt.Errorf("invalid CiPipelineId %d", ciPipelineId), nil, http.StatusBadRequest)
		return
	} else if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	token := r.Header.Get("token")
	// RBAC enforcer applying
	resourceName := handler.enforcerUtil.GetAppRBACName(ciPipeline.App.AppName)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, resourceName); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	// RBAC enforcer Ends
	resp, err := handler.ciCdPipelineOrchestrator.GetSourceCiDownStreamFilters(r.Context(), ciPipelineId)
	if err != nil {
		common.WriteJsonResp(w, fmt.Errorf("error getting environment info for given source Ci pipeline id"), "error getting environment info for given source Ci pipeline id", http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetSourceCiDownStreamInfo(w http.ResponseWriter, r *http.Request) {
	decoder := schema.NewDecoder()
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	ciPipelineId, err := strconv.Atoi(vars["ciPipelineId"])
	if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	req := &bean2.SourceCiDownStreamFilters{}
	err = decoder.Decode(req, r.URL.Query())
	if err != nil {
		handler.Logger.Errorw("request err, GetSourceCiDownStreamInfo", "err", err, "payload", req)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	// Convert searchKey to lowercase
	req.SearchKey = strings.ToLower(req.SearchKey)
	req.SortBy = pagination.AppName
	if req.Size == 0 {
		req.Size = 20
	}
	if len(req.SortOrder) == 0 {
		req.SortOrder = pagination.Asc
	}
	token := r.Header.Get("token")
	ciPipeline, err := handler.ciPipelineRepository.FindOneWithAppData(ciPipelineId)
	if util.IsErrNoRows(err) {
		common.WriteJsonResp(w, fmt.Errorf("invalid CiPipelineId %d", ciPipelineId), nil, http.StatusBadRequest)
		return
	} else if err != nil {
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	// RBAC enforcer applying
	resourceName := handler.enforcerUtil.GetAppRBACName(ciPipeline.App.AppName)
	if ok := handler.enforcer.Enforce(token, casbin.ResourceApplications, casbin.ActionGet, resourceName); !ok {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusForbidden)
		return
	}
	// RBAC enforcer Ends
	linkedCIDetails, err := handler.ciCdPipelineOrchestrator.GetSourceCiDownStreamInfo(r.Context(), ciPipelineId, req)
	if err != nil {
		handler.Logger.Errorw("service err, PatchCiPipelines", "err", err, "ciPipelineId", ciPipelineId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, err, linkedCIDetails, http.StatusOK)
}

func (handler *PipelineConfigRestHandlerImpl) GetAppMetadataListByEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := r.Header.Get("token")
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if userId == 0 || err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	envId, err := strconv.Atoi(vars["envId"])
	if err != nil {
		handler.Logger.Errorw("request err, GetAppMetadataListByEnvironment", "err", err, "envId", envId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	v := r.URL.Query()
	appIdsString := v.Get("appIds")
	var appIds []int
	if len(appIdsString) > 0 {
		appIdsSlices := strings.Split(appIdsString, ",")
		for _, appId := range appIdsSlices {
			id, err := strconv.Atoi(appId)
			if err != nil {
				common.WriteJsonResp(w, err, "please provide valid appIds", http.StatusBadRequest)
				return
			}
			appIds = append(appIds, id)
		}
	}

	resp, err := handler.pipelineBuilder.GetAppMetadataListByEnvironment(envId, appIds)
	if err != nil {
		handler.Logger.Errorw("service err, GetAppMetadataListByEnvironment", "envId", envId, "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	// return all if user is super admin
	if isActionUserSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); isActionUserSuperAdmin {
		common.WriteJsonResp(w, err, resp, http.StatusOK)
		return
	}

	// get all the appIds
	appIds = make([]int, 0)
	appContainers := resp.Apps
	for _, appBean := range resp.Apps {
		appIds = append(appIds, appBean.AppId)
	}

	// get rbac objects for the appids
	rbacObjectsWithAppId := handler.enforcerUtil.GetRbacObjectsByAppIds(appIds)
	rbacObjects := maps.Values(rbacObjectsWithAppId)
	// enforce rbac in batch
	rbacResult := handler.enforcer.EnforceInBatch(token, casbin.ResourceApplications, casbin.ActionGet, rbacObjects)
	// filter out rbac passed apps
	resp.Apps = make([]*bean1.AppMetaData, 0)
	for _, appBean := range appContainers {
		rbacObject := rbacObjectsWithAppId[appBean.AppId]
		if rbacResult[rbacObject] {
			resp.Apps = append(resp.Apps, appBean)
		}
	}
	resp.AppCount = len(resp.Apps)
	common.WriteJsonResp(w, err, resp, http.StatusOK)
}
