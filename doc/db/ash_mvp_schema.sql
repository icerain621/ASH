SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  uid VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(128) NOT NULL,
  email VARCHAR(128) NULL,
  avatar_url VARCHAR(512) NULL,
  status TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS spaces (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  owner_id BIGINT NOT NULL,
  description VARCHAR(512) NULL,
  visibility VARCHAR(16) NOT NULL DEFAULT 'private',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_owner_id (owner_id),
  CONSTRAINT fk_spaces_owner FOREIGN KEY (owner_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS space_members (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  space_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  role VARCHAR(16) NOT NULL,
  joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status TINYINT NOT NULL DEFAULT 1,
  UNIQUE KEY uk_space_user (space_id, user_id),
  KEY idx_user_id (user_id),
  CONSTRAINT fk_space_members_space FOREIGN KEY (space_id) REFERENCES spaces(id),
  CONSTRAINT fk_space_members_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tasks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(256) NOT NULL,
  goal TEXT NOT NULL,
  status VARCHAR(24) NOT NULL,
  mode VARCHAR(16) NOT NULL DEFAULT 'assist',
  project_id VARCHAR(64) NOT NULL,
  space_id BIGINT NULL,
  creator_id BIGINT NOT NULL,
  priority VARCHAR(16) NOT NULL DEFAULT 'medium',
  risk_level VARCHAR(16) NOT NULL DEFAULT 'low',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_project_status (project_id, status),
  KEY idx_creator_id (creator_id),
  KEY idx_space_id (space_id),
  CONSTRAINT fk_tasks_space FOREIGN KEY (space_id) REFERENCES spaces(id),
  CONSTRAINT fk_tasks_creator FOREIGN KEY (creator_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS agent_runs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  task_id BIGINT NOT NULL,
  state VARCHAR(24) NOT NULL,
  plan_json JSON NULL,
  risk_flags_json JSON NULL,
  started_at DATETIME NULL,
  ended_at DATETIME NULL,
  created_by BIGINT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_task_state (task_id, state),
  KEY idx_created_by (created_by),
  CONSTRAINT fk_agent_runs_task FOREIGN KEY (task_id) REFERENCES tasks(id),
  CONSTRAINT fk_agent_runs_creator FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS task_steps (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_id BIGINT NOT NULL,
  task_id BIGINT NOT NULL,
  step_no INT NOT NULL,
  step_type VARCHAR(32) NOT NULL,
  status VARCHAR(24) NOT NULL,
  input_json JSON NULL,
  output_json JSON NULL,
  error_message TEXT NULL,
  started_at DATETIME NULL,
  ended_at DATETIME NULL,
  UNIQUE KEY uk_run_step (run_id, step_no),
  KEY idx_task_status (task_id, status),
  CONSTRAINT fk_task_steps_run FOREIGN KEY (run_id) REFERENCES agent_runs(id),
  CONSTRAINT fk_task_steps_task FOREIGN KEY (task_id) REFERENCES tasks(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS memories (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  scope VARCHAR(16) NOT NULL,
  owner_user_id BIGINT NULL,
  project_id VARCHAR(64) NULL,
  session_id VARCHAR(64) NULL,
  content TEXT NOT NULL,
  summary VARCHAR(512) NULL,
  tags_json JSON NULL,
  source_type VARCHAR(32) NOT NULL,
  source_ref VARCHAR(128) NULL,
  confidence DECIMAL(5,4) NOT NULL DEFAULT 0.7000,
  is_pinned TINYINT NOT NULL DEFAULT 0,
  ttl_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_scope_owner (scope, owner_user_id, project_id),
  KEY idx_ttl (ttl_at),
  CONSTRAINT fk_memories_owner FOREIGN KEY (owner_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS memory_events (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  memory_id BIGINT NOT NULL,
  event_type VARCHAR(24) NOT NULL,
  delta_json JSON NULL,
  actor_id BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_memory_event (memory_id, event_type),
  KEY idx_actor_id (actor_id),
  CONSTRAINT fk_memory_events_memory FOREIGN KEY (memory_id) REFERENCES memories(id),
  CONSTRAINT fk_memory_events_actor FOREIGN KEY (actor_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tool_calls (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_id BIGINT NOT NULL,
  step_id BIGINT NULL,
  tool_name VARCHAR(64) NOT NULL,
  request_json JSON NULL,
  response_json JSON NULL,
  status VARCHAR(16) NOT NULL,
  latency_ms INT NOT NULL DEFAULT 0,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_run_tool (run_id, tool_name),
  KEY idx_step_id (step_id),
  CONSTRAINT fk_tool_calls_run FOREIGN KEY (run_id) REFERENCES agent_runs(id),
  CONSTRAINT fk_tool_calls_step FOREIGN KEY (step_id) REFERENCES task_steps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS feedback_records (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  target_type VARCHAR(24) NOT NULL,
  target_id VARCHAR(64) NOT NULL,
  rating TINYINT NOT NULL,
  category VARCHAR(32) NOT NULL,
  comment VARCHAR(1024) NULL,
  creator_id BIGINT NOT NULL,
  project_id VARCHAR(64) NULL,
  space_id BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_target (target_type, target_id),
  KEY idx_created (created_at),
  KEY idx_creator_id (creator_id),
  CONSTRAINT fk_feedback_creator FOREIGN KEY (creator_id) REFERENCES users(id),
  CONSTRAINT fk_feedback_space FOREIGN KEY (space_id) REFERENCES spaces(id),
  CONSTRAINT chk_feedback_rating CHECK (rating BETWEEN 1 AND 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET FOREIGN_KEY_CHECKS = 1;
