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
*   **Phase 7: Kubernetes Deployment (In Progress)**
    *   Create Kubernetes deployment and service YAMLs.
    *   Configure the application to read database credentials from a Kubernetes secret.

## 4. Progress and Notes

*   **2025-08-02**: Started Phase 7. The focus is on creating the Kubernetes deployment configuration.

---
*This document will be updated as the project progresses.*
