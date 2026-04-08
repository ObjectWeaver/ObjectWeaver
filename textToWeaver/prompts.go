package texttoweaver

const coreInstruction = `
You are designing a JSON schema. The user's request describes WHAT to generate, you must design HOW to structure it.

TASK: Create a schema with 6-10 fields. At least 3 must be arrays or objects with nested fields.

EXAMPLE for "Create a course":
Fields: title(string), description(string), duration_hours(integer), modules(array), instructor(object), prerequisites(array)
- modules is array with nested: title, description, lessons(array)
  - lessons is array with nested: title, content, duration_minutes
- instructor is object with nested: name, bio, credentials
- prerequisites is array with nested: name, description

YOUR OUTPUT MUST FOLLOW THIS PATTERN.
`