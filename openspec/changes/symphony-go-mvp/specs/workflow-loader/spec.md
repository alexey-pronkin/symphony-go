## ADDED Requirements

### Requirement: Load WORKFLOW.md from path
The system SHALL load a WORKFLOW.md file given an explicit path or default to `./WORKFLOW.md` in the process working directory. It SHALL return a parsed `WorkflowDefinition` containing `config` (front matter map) and `prompt_template` (trimmed Markdown body).

#### Scenario: Explicit path loads successfully
- **WHEN** a valid WORKFLOW.md path is provided and the file exists
- **THEN** the loader returns a WorkflowDefinition with non-nil config and prompt_template

#### Scenario: Default path used when none provided
- **WHEN** no explicit path is given
- **THEN** the loader reads from `./WORKFLOW.md`

#### Scenario: Missing file returns typed error
- **WHEN** the specified file does not exist
- **THEN** the loader returns a `missing_workflow_file` error

### Requirement: Parse YAML front matter and prompt body
The system SHALL parse files starting with `---` as having YAML front matter up to the closing `---`, with the remainder as the prompt body. Files without front matter SHALL use an empty config map and the full content as the prompt body.

#### Scenario: Front matter parses to map
- **WHEN** the file has valid YAML front matter
- **THEN** config is a non-nil map with the parsed keys

#### Scenario: Non-map front matter is an error
- **WHEN** the YAML front matter decodes to a non-map type
- **THEN** the loader returns a `workflow_front_matter_not_a_map` error

#### Scenario: Invalid YAML returns parse error
- **WHEN** the front matter contains invalid YAML
- **THEN** the loader returns a `workflow_parse_error` error

#### Scenario: Prompt body is trimmed
- **WHEN** the Markdown body has leading/trailing whitespace
- **THEN** the returned prompt_template has whitespace trimmed

### Requirement: Dynamic file-watch reload
The system SHALL watch the loaded WORKFLOW.md file for changes and re-read and re-apply config and prompt without restart.

#### Scenario: Changed file triggers reload
- **WHEN** the WORKFLOW.md file is modified on disk
- **THEN** the orchestrator applies the new config values for future operations

#### Scenario: Invalid reload keeps last good config
- **WHEN** a reloaded file contains invalid YAML
- **THEN** the system emits an operator-visible error and continues with the last valid config

### Requirement: Template rendering with strict variables
The system SHALL render the prompt template with `issue` and `attempt` variables. Unknown variables SHALL cause a render failure.

#### Scenario: Template renders issue fields
- **WHEN** a template references `{{ .issue.title }}` and the issue has a title
- **THEN** the rendered prompt contains the issue title

#### Scenario: Unknown variable fails rendering
- **WHEN** a template references an undefined variable
- **THEN** rendering returns a `template_render_error`
