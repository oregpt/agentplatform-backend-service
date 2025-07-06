-- Current Spanner Database Schema for AgentPlatform
-- Last updated: 2025-07-06

CREATE TABLE Agents (
  AgentID STRING(36) NOT NULL,
  OrganizationID STRING(36),
  Name STRING(255) NOT NULL,
  Description STRING(MAX),
  Instructions STRING(MAX),
  AIProvider STRING(50) NOT NULL,
  CreatedBy STRING(128) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(AgentID);

CREATE TABLE Files (
  AgentID STRING(36) NOT NULL,
  FileID STRING(36) NOT NULL,
  Name STRING(255) NOT NULL,
  Path STRING(MAX) NOT NULL,
  ContentType STRING(100) NOT NULL,
  SizeBytes INT64 NOT NULL,
  CreatedBy STRING(128) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(AgentID, FileID),
  INTERLEAVE IN PARENT Agents ON DELETE CASCADE;

CREATE TABLE Organizations (
  OrganizationID STRING(36) NOT NULL,
  Name STRING(255) NOT NULL,
  Description STRING(MAX),
  CreatedBy STRING(128) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(OrganizationID);

CREATE TABLE UserOrgs (
  OrganizationID STRING(36) NOT NULL,
  UserID STRING(128) NOT NULL,
  Email STRING(255) NOT NULL,
  DisplayName STRING(255),
  Role STRING(50) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(OrganizationID, UserID),
  INTERLEAVE IN PARENT Organizations ON DELETE CASCADE;

CREATE TABLE UserAgents (
  OrganizationID STRING(36) NOT NULL,
  UserID STRING(128) NOT NULL,
  AgentID STRING(36) NOT NULL,
  CreatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(OrganizationID, UserID, AgentID),
  INTERLEAVE IN PARENT UserOrgs ON DELETE CASCADE;

CREATE TABLE Users (
  UserID STRING(36) NOT NULL,
  Email STRING(255) NOT NULL,
  DisplayName STRING(255),
  Address STRING(1024),
  Phone STRING(50),
  Metadata JSON,
  CreatedAt TIMESTAMP NOT NULL,
  UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY(UserID);

ALTER TABLE UserOrgs ADD CONSTRAINT FK_UserOrgs_Users FOREIGN KEY(UserID) REFERENCES Users(UserID);

-- Schema Notes:
-- 1. Agents table has no foreign key constraint on OrganizationID, allowing null/empty values
-- 2. Files are interleaved in Agents (parent-child relationship)
-- 3. UserOrgs are interleaved in Organizations
-- 4. UserAgents are interleaved in UserOrgs
-- 5. There is a foreign key from UserOrgs.UserID to Users.UserID
