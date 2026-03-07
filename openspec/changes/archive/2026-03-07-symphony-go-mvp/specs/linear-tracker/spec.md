## ADDED Requirements

### Requirement: Fetch candidate issues
The system SHALL query Linear GraphQL for issues in the configured active states for the configured project slug, paginate all results (page size 50), and return normalized Issue records.

#### Scenario: Returns issues in active states
- **WHEN** the project has issues in `In Progress` state
- **THEN** the client returns those issues as normalized Issue structs

#### Scenario: Pagination collects all pages
- **WHEN** there are more than 50 matching issues
- **THEN** all issues across pages are returned in order

#### Scenario: Empty result returns empty slice
- **WHEN** no issues match the active states
- **THEN** the client returns an empty slice without error

### Requirement: Fetch issue states by IDs
The system SHALL query Linear for current states of specific issue IDs using `[ID!]` typing.

#### Scenario: Returns current state for each ID
- **WHEN** a list of issue IDs is provided
- **THEN** the client returns normalized issues with current state field

#### Scenario: Empty ID list returns empty without API call
- **WHEN** an empty ID list is provided
- **THEN** the client returns immediately without making a network request

### Requirement: Normalize issue fields
The system SHALL produce Issue records with all fields from SPEC.md Section 4.1.1: labels lowercased, blockers from inverse `blocks` relations, priority as integer or null, timestamps parsed from ISO-8601.

#### Scenario: Labels are lowercased
- **WHEN** a Linear issue has label `Backend`
- **THEN** the normalized issue has label `backend`

#### Scenario: Blocker derived from blocks relation
- **WHEN** issue B has a relation type `blocks` toward issue A
- **THEN** issue A's `blocked_by` list includes a reference to B

### Requirement: Linear error categorization
The system SHALL categorize errors as: `linear_api_request` (transport), `linear_api_status` (non-200 HTTP), `linear_graphql_errors` (top-level errors field), `linear_unknown_payload` (unexpected shape).

#### Scenario: Non-200 response returns api_status error
- **WHEN** Linear returns HTTP 503
- **THEN** the error is categorized as `linear_api_status`

#### Scenario: GraphQL errors field returns graphql error
- **WHEN** the response body contains top-level `errors`
- **THEN** the error is categorized as `linear_graphql_errors`
