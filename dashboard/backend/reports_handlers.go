package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// VirtualServiceUsage represents an application that uses a virtual service.
type VirtualServiceUsage struct {
	AppID   int    `json:"appId"`
	AppName string `json:"appName"`
}

// App represents a simple application with its ID and name.
type App struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BuildTime represents the build time for a CI pipeline.
type BuildTime struct {
	AppName    string        `json:"appName"`
	PipelineID int           `json:"pipelineId"`
	BuildTime  time.Duration `json:"buildTime"`
}

// ScopedVariableUsage represents an entity that uses a scoped variable.
type ScopedVariableUsage struct {
	EntityType int `json:"entityType"`
	EntityID   int `json:"entityId"`
}

// DeployedBranch represents a deployed branch for an application in an environment.
type DeployedBranch struct {
	AppName string `json:"appName"`
	Branch  string `json:"branch"`
}

// Pipeline represents a CI pipeline.
type Pipeline struct {
	ID      int    `json:"id"`
	AppName string `json:"appName"`
	Name    string `json:"name"`
}

// TrivyScanResult represents a single Trivy scan result.
type TrivyScanResult struct {
	CVE        string `json:"cve"`
	Severity   int    `json:"severity"`
	Package    string `json:"package"`
	Version    string `json:"version"`
	FixedVersion string `json:"fixedVersion"`
}

// ConfigDump represents a dump of secrets and configmaps.
type ConfigDump struct {
	AppName     string `json:"appName"`
	EnvName     string `json:"envName"`
	ConfigMap   string `json:"configMap"`
	Secret      string `json:"secret"`
}


func getVirtualServiceUsageHandler(w http.ResponseWriter, r *http.Request) {
	// Query to fetch deployment templates
	rows, err := db.Query(`
		SELECT
			a.id,
			a.app_name,
			c.values_yaml
		FROM
			app a
		JOIN
			charts c ON a.id = c.app_id
		WHERE
			c.latest = true;
	`)
	if err != nil {
		http.Error(w, "Failed to query charts", http.StatusInternalServerError)
		log.Printf("Failed to query charts: %v", err)
		return
	}
	defer rows.Close()

	var results []VirtualServiceUsage
	for rows.Next() {
		var appID int
		var appName string
		var valuesYaml string
		if err := rows.Scan(&appID, &appName, &valuesYaml); err != nil {
			http.Error(w, "Failed to scan chart data", http.StatusInternalServerError)
			log.Printf("Failed to scan chart data: %v", err)
			return
		}

		var values map[string]interface{}
		if err := yaml.Unmarshal([]byte(valuesYaml), &values); err != nil {
			// Not all values_yaml will be valid yaml, so we just log the error and continue
			log.Printf("Failed to unmarshal values_yaml for app %d: %v", appID, err)
			continue
		}

		// This is a very basic check. A more robust implementation would
		// traverse the values map to find the 'kind: VirtualService'.
		if _, ok := values["kind"]; ok && values["kind"] == "VirtualService" {
			results = append(results, VirtualServiceUsage{AppID: appID, AppName: appName})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode results to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode results to JSON: %v", err)
	}
}

func getAppsNotDeployedHandler(w http.ResponseWriter, r *http.Request) {
	envIDsStr := r.URL.Query().Get("envIds")
	if envIDsStr == "" {
		http.Error(w, "envIds query parameter is required", http.StatusBadRequest)
		return
	}
	envIDs := strings.Split(envIDsStr, ",")

	// Query to fetch apps not deployed in the given environments
	// This query is a bit complex. It finds all apps and then subtracts the ones that are deployed in the given environments.
	query := `
		SELECT
			id,
			app_name
		FROM
			app
		WHERE
			id NOT IN (
				SELECT
					app_id
				FROM
					installed_apps
				WHERE
					environment_id = ANY($1)
			);
	`
	rows, err := db.Query(query, envIDs)
	if err != nil {
		http.Error(w, "Failed to query apps not deployed", http.StatusInternalServerError)
		log.Printf("Failed to query apps not deployed: %v", err)
		return
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		if err := rows.Scan(&app.ID, &app.Name); err != nil {
			http.Error(w, "Failed to scan app data", http.StatusInternalServerError)
			log.Printf("Failed to scan app data: %v", err)
			return
		}
		apps = append(apps, app)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, "Failed to encode apps to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode apps to JSON: %v", err)
	}
}

func getBuildTimeAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("appId")
	query := `
		SELECT
			a.app_name,
			cw.ci_pipeline_id,
			(cw.finished_on - cw.started_on) as build_time
		FROM
			ci_workflow cw
		JOIN
			ci_pipeline cp ON cw.ci_pipeline_id = cp.id
		JOIN
			app a ON cp.app_id = a.id
		WHERE
			cw.status = 'Succeeded'
	`
	var rows *sql.Rows
	var err error
	if appID != "" {
		query += " AND a.id = $1"
		rows, err = db.Query(query, appID)
	} else {
		rows, err = db.Query(query)
	}

	if err != nil {
		http.Error(w, "Failed to query build time analytics", http.StatusInternalServerError)
		log.Printf("Failed to query build time analytics: %v", err)
		return
	}
	defer rows.Close()

	var buildTimes []BuildTime
	for rows.Next() {
		var bt BuildTime
		if err := rows.Scan(&bt.AppName, &bt.PipelineID, &bt.BuildTime); err != nil {
			http.Error(w, "Failed to scan build time data", http.StatusInternalServerError)
			log.Printf("Failed to scan build time data: %v", err)
			return
		}
		buildTimes = append(buildTimes, bt)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buildTimes); err != nil {
		http.Error(w, "Failed to encode build times to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode build times to JSON: %v", err)
	}
}

func getChartVersionUsageHandler(w http.ResponseWriter, r *http.Request) {
	chartVersion := r.URL.Query().Get("chartVersion")
	if chartVersion == "" {
		http.Error(w, "chartVersion query parameter is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			a.id,
			a.app_name
		FROM
			app a
		JOIN
			charts c ON a.id = c.app_id
		WHERE
			c.chart_version = $1
			AND c.latest = true;
	`
	rows, err := db.Query(query, chartVersion)
	if err != nil {
		http.Error(w, "Failed to query chart version usage", http.StatusInternalServerError)
		log.Printf("Failed to query chart version usage: %v", err)
		return
	}
	defer rows.Close()

	var apps []App
	for rows.Next() {
		var app App
		if err := rows.Scan(&app.ID, &app.Name); err != nil {
			http.Error(w, "Failed to scan app data", http.StatusInternalServerError)
			log.Printf("Failed to scan app data: %v", err)
			return
		}
		apps = append(apps, app)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, "Failed to encode apps to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode apps to JSON: %v", err)
	}
}

func getScopedVariableUsageHandler(w http.ResponseWriter, r *http.Request) {
	variableName := r.URL.Query().Get("variableName")
	if variableName == "" {
		http.Error(w, "variableName query parameter is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			entity_type,
			entity_id
		FROM
			variable_entity_mapping
		WHERE
			variable_name = $1
			AND is_deleted = false;
	`
	rows, err := db.Query(query, variableName)
	if err != nil {
		http.Error(w, "Failed to query scoped variable usage", http.StatusInternalServerError)
		log.Printf("Failed to query scoped variable usage: %v", err)
		return
	}
	defer rows.Close()

	var usages []ScopedVariableUsage
	for rows.Next() {
		var usage ScopedVariableUsage
		if err := rows.Scan(&usage.EntityType, &usage.EntityID); err != nil {
			http.Error(w, "Failed to scan scoped variable usage data", http.StatusInternalServerError)
			log.Printf("Failed to scan scoped variable usage data: %v", err)
			return
		}
		usages = append(usages, usage)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(usages); err != nil {
		http.Error(w, "Failed to encode usages to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode usages to JSON: %v", err)
	}
}

func getDeployedBranchHandler(w http.ResponseWriter, r *http.Request) {
	envID := r.URL.Query().Get("envId")
	if envID == "" {
		http.Error(w, "envId query parameter is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			a.app_name,
			cpm.value
		FROM
			app a
		JOIN
			ci_pipeline cp ON a.id = cp.app_id
		JOIN
			ci_pipeline_material cpm ON cp.id = cpm.ci_pipeline_id
		JOIN
			pipeline p ON cp.id = p.ci_pipeline_id
		WHERE
			p.environment_id = $1
			AND cpm.type = 'SOURCE_TYPE_BRANCH_FIXED';
	`
	rows, err := db.Query(query, envID)
	if err != nil {
		http.Error(w, "Failed to query deployed branches", http.StatusInternalServerError)
		log.Printf("Failed to query deployed branches: %v", err)
		return
	}
	defer rows.Close()

	var deployedBranches []DeployedBranch
	for rows.Next() {
		var db DeployedBranch
		if err := rows.Scan(&db.AppName, &db.Branch); err != nil {
			http.Error(w, "Failed to scan deployed branch data", http.StatusInternalServerError)
			log.Printf("Failed to scan deployed branch data: %v", err)
			return
		}
		deployedBranches = append(deployedBranches, db)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployedBranches); err != nil {
		http.Error(w, "Failed to encode deployed branches to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode deployed branches to JSON: %v", err)
	}
}

func getNoImageScanReportHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT
			cp.id,
			a.app_name,
			cp.name
		FROM
			ci_pipeline cp
		JOIN
			app a ON cp.app_id = a.id
		WHERE
			cp.scan_enabled = false;
	`
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to query pipelines with no image scan", http.StatusInternalServerError)
		log.Printf("Failed to query pipelines with no image scan: %v", err)
		return
	}
	defer rows.Close()

	var pipelines []Pipeline
	for rows.Next() {
		var p Pipeline
		if err := rows.Scan(&p.ID, &p.AppName, &p.Name); err != nil {
			http.Error(w, "Failed to scan pipeline data", http.StatusInternalServerError)
			log.Printf("Failed to scan pipeline data: %v", err)
			return
		}
		pipelines = append(pipelines, p)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pipelines); err != nil {
		http.Error(w, "Failed to encode pipelines to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode pipelines to JSON: %v", err)
	}
}

func exportTrivyScanResultHandler(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("appId")
	if appID == "" {
		http.Error(w, "appId query parameter is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			cs.name,
			cs.severity,
			cs.package,
			cs.version,
			cs.fixed_version
		FROM
			cve_store cs
		JOIN
			image_scan_execution_result iser ON cs.name = iser.cve_store_name
		JOIN
			image_scan_execution_history iseh ON iser.image_scan_execution_history_id = iseh.id
		JOIN
			image_scan_deploy_info isdi ON iseh.id = ANY(isdi.image_scan_execution_history_id)
		WHERE
			isdi.scan_object_meta_id = $1;

	`
	rows, err := db.Query(query, appID)
	if err != nil {
		http.Error(w, "Failed to query trivy scan results", http.StatusInternalServerError)
		log.Printf("Failed to query trivy scan results: %v", err)
		return
	}
	defer rows.Close()

	var results []TrivyScanResult
	for rows.Next() {
		var res TrivyScanResult
		if err := rows.Scan(&res.CVE, &res.Severity, &res.Package, &res.Version, &res.FixedVersion); err != nil {
			http.Error(w, "Failed to scan trivy scan result data", http.StatusInternalServerError)
			log.Printf("Failed to scan trivy scan result data: %v", err)
			return
		}
		results = append(results, res)
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=trivy-scan-results.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"CVE", "Severity", "Package", "Version", "FixedVersion"}
	if err := writer.Write(header); err != nil {
		log.Printf("Failed to write CSV header: %v", err)
		return
	}

	// Write rows
	for _, res := range results {
		row := []string{
			res.CVE,
			strconv.Itoa(res.Severity),
			res.Package,
			res.Version,
			res.FixedVersion,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Failed to write CSV row: %v", err)
			return
		}
	}
}

func dumpSecretsAndConfigMapsHandler(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("clusterId")
	if clusterID == "" {
		http.Error(w, "clusterId query parameter is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT
			a.app_name,
			e.environment_name,
			cmel.config_map_data,
			cmel.secret_data
		FROM
			config_map_env_level cmel
		JOIN
			app a ON cmel.app_id = a.id
		JOIN
			environment e ON cmel.environment_id = e.id
		WHERE
			e.cluster_id = $1;
	`
	rows, err := db.Query(query, clusterID)
	if err != nil {
		http.Error(w, "Failed to query secrets and configmaps", http.StatusInternalServerError)
		log.Printf("Failed to query secrets and configmaps: %v", err)
		return
	}
	defer rows.Close()

	var results []ConfigDump
	for rows.Next() {
		var dump ConfigDump
		if err := rows.Scan(&dump.AppName, &dump.EnvName, &dump.ConfigMap, &dump.Secret); err != nil {
			http.Error(w, "Failed to scan secrets and configmaps data", http.StatusInternalServerError)
			log.Printf("Failed to scan secrets and configmaps data: %v", err)
			return
		}
		results = append(results, dump)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode results to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode results to JSON: %v", err)
	}
}
