# Devtron Universal Data Querying & Auditing Dashboard

This document provides an overview of the Devtron Universal Data Querying & Auditing Dashboard project. It is intended to be a reference for developers and stakeholders.

## 1. Project Goal

The primary goal of this project is to build a standalone, read-only dashboard for querying and auditing data from the Devtron application's PostgreSQL database. The dashboard will provide insights into deployments, user management, application configurations, and system-wide auditing, with a strong focus on data export to CSV.

## 2. Core Principles

The development of this dashboard follows these core principles:

*   **Zero Code Modification**: The dashboard is a completely separate application and does not modify the existing Devtron codebase.
*   **Read-Only Database Access**: The dashboard only performs `SELECT` queries on the database and never modifies data.
*   **Independent Deployment**: The dashboard is containerized and deployed independently from the main Devtron application.
*   **Maintainability**: The dashboard is designed for easy updates and future feature additions.

## 3. High-Level Implementation Plan

The implementation is divided into the following phases:

*   **Phase 1: Foundation (Complete)**
*   **Phase 2: Core Querying & User Management (Complete)**
*   **Phase 3: Deployment & Application Insights (Complete)**
*   **Phase 4: Auditing & Advanced Querying (Complete)**
*   **Phase 5: Finalization & Documentation (Complete)**
*   **Phase 6: Advanced Auditing Features (Complete)**

## 4. Progress and Notes

*   **2025-08-02**: Completed Phase 6. All requested advanced auditing features are implemented.
*   **Testing Status**: The Go backend has been tested by building it and running a unit test for the `/health` endpoint. However, due to persistent environment issues (Docker permission errors, `getwd` error in Go), I was unable to build or run the Docker container to test the full application.

---
*This document will be updated as the project progresses.*
