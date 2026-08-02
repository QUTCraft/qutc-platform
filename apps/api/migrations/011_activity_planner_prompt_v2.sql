UPDATE agent_definitions
SET system_policy_version = 'activity-planner/v2',
    updated_at = CURRENT_TIMESTAMP(3)
WHERE `key` = 'activity-planner'
  AND system_policy_version <> 'activity-planner/v2';
