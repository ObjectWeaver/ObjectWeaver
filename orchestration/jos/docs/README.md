# JOS Documentation Index

Welcome to the JOS (JSON Object Schema) generation system documentation. This index will help you find the right document for your needs.

## 📚 Documentation Files

### 1. **ARCHITECTURE_FLOW.md** - Complete System Flow
**When to use**: Understanding how the system works from initialization to generation

**Contents**:
- Complete initialization flow with file names and line numbers
- Detailed generation flow (10 phases)
- Component dependency graph
- Data flow examples
- Entry point tracking (HTTP, gRPC)

**Best for**:
- New team members learning the system
- Debugging complex flows
- Understanding component interactions
- Finding where things get created and injected

---

### 2. **INTERFACE_IMPLEMENTATION_MAP.md** - Interface → Code Mapping
**When to use**: Finding implementations of interfaces

**Contents**:
- Every interface with its implementation(s)
- File paths and line numbers
- Purpose and responsibilities of each interface
- Quick navigation tips

**Best for**:
- "Where is this interface implemented?"
- Finding concrete classes from interface names
- Understanding implementation options
- Jumping between interface and code

---

### 3. **QUICK_REFERENCE.md** - Developer Cheat Sheet
**When to use**: Quick lookups and common tasks

**Contents**:
- Quick reference tables
- Common tasks (adding processors, strategies, modes)
- Code templates
- Debugging tips
- Configuration options
- File naming conventions

**Best for**:
- Daily development work
- Adding new components
- Quick lookups
- Common patterns and idioms

---

### 4. **REFACTORING_PLAN.md** - Future Improvements
**When to use**: Planning architectural changes

**Contents**:
- Analysis of current structure
- Proposed refactoring approach
- Benefits and tradeoffs
- Migration strategy

**Best for**:
- Understanding design decisions
- Planning future improvements
- Historical context

---

## 🎯 Quick Navigation

### I want to...

**Understand how the system works**
→ Start with **ARCHITECTURE_FLOW.md**

**Find where an interface is implemented**
→ Use **INTERFACE_IMPLEMENTATION_MAP.md**

**Add a new feature**
→ Check **QUICK_REFERENCE.md** → "Common Tasks" section

**Debug an issue**
→ Use **ARCHITECTURE_FLOW.md** → "Generation Flow" + **QUICK_REFERENCE.md** → "Debugging Tips"

**See all interfaces at once**
→ Open `domain/interfaces.go` (all interfaces are there with detailed comments)

**Find a specific component**
→ Use **INTERFACE_IMPLEMENTATION_MAP.md** → "Summary Table"

---

## 🏗️ Architecture Overview

### Layers

```
┌─────────────────────────────────────────┐
│     Service Layer (HTTP/gRPC)           │  service/, grpcService/
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│        Factory (DI)                      │  factory/
└──────────────────┬──────────────────────┘
                   │
┌──────────────────▼──────────────────────┐
│   Application (Generators)               │  application/
└────┬────────────┬────────────┬──────────┘
     │            │            │
  ┌──▼──┐     ┌──▼──┐     ┌──▼──┐
  │Sync │     │Field│     │Token│
  │     │     │Stream│    │Stream│
  └─────┘     └─────┘     └─────┘
     │            │            │
     │            ▼            │
     │    ┌───────────────┐   │
     └────►   Domain      ◄───┘  domain/
          │  (Interfaces) │
          └───────┬───────┘
                  │
     ┌────────────┼────────────┐
     │            │            │
  ┌──▼────┐  ┌───▼───┐  ┌────▼────┐
  │Analysis│  │Execute│  │Strategy │
  └────────┘  └───┬───┘  └─────────┘
                  │
          ┌───────┼────────┐
          │       │        │
      ┌───▼──┐ ┌─▼──┐ ┌───▼────┐
      │ LLM  │ │Prompt│ │Assembly│
      └──────┘ └─────┘ └────────┘
```

### Design Principles

1. **Dependency Injection**: All components receive dependencies via constructor
2. **Interface Segregation**: Small, focused interfaces
3. **Dependency Inversion**: Depend on abstractions (interfaces), not concrete types
4. **Single Responsibility**: Each component has one job
5. **Open/Closed**: Easy to extend (new processors, strategies) without modifying existing code

---

## 📖 Core Concepts

### Generator
The main API - takes a schema and prompt, returns generated JSON.

**Types**:
- **Default**: Synchronous, returns complete result
- **Streaming**: Streams fields as they complete
- **Progressive**: Streams tokens as they arrive

### SchemaAnalyzer
Breaks down complex schemas into processable fields with dependencies.

### TaskExecutor
Routes field generation to appropriate type-specific processors.

### TypeProcessor
Handles generation for specific JSON types (string, object, array, etc.).

### ExecutionStrategy
Controls when and how tasks execute (sequential, parallel, dependency-aware).

### ResultAssembler
Collects task results and builds final JSON object.

---

## 🔍 Finding Your Way

### By Role

**New Developer**:
1. Read: ARCHITECTURE_FLOW.md (Overview)
2. Browse: INTERFACE_IMPLEMENTATION_MAP.md (What's available)
3. Bookmark: QUICK_REFERENCE.md (Daily reference)

**Adding Features**:
1. Check: QUICK_REFERENCE.md → "Common Tasks"
2. Reference: INTERFACE_IMPLEMENTATION_MAP.md → Find similar code
3. Follow: Code templates in QUICK_REFERENCE.md

**Debugging**:
1. Trace flow: ARCHITECTURE_FLOW.md → "Generation Flow"
2. Add logging: QUICK_REFERENCE.md → "Debugging Tips"
3. Check implementation: INTERFACE_IMPLEMENTATION_MAP.md

**Refactoring**:
1. Understand current: ARCHITECTURE_FLOW.md
2. Review plan: REFACTORING_PLAN.md
3. Check interfaces: domain/interfaces.go

---

## 💡 Key Files

| File | Purpose | When to Look |
|------|---------|--------------|
| `domain/interfaces.go` | All interface definitions | Understanding contracts |
| `factory/generator_factory.go` | Component creation & wiring | Seeing what gets injected |
| `application/default_generator.go` | Main generation workflow | Understanding process flow |
| `infrastructure/execution/task_executor.go` | Task routing | Understanding type handling |
| `service/objectGen.go` | HTTP entry point | Seeing system usage |

---

## 🚀 Getting Started

### 1. First Time Here?
Read **ARCHITECTURE_FLOW.md** sections:
- Overview
- Initialization Flow
- Generation Flow (just the diagram)

### 2. Ready to Code?
1. Pick your task from **QUICK_REFERENCE.md**
2. Find similar code in **INTERFACE_IMPLEMENTATION_MAP.md**
3. Follow the template

### 3. Need Details?
- **How does X work?** → ARCHITECTURE_FLOW.md
- **Where is X implemented?** → INTERFACE_IMPLEMENTATION_MAP.md
- **How do I add X?** → QUICK_REFERENCE.md

---

## 📝 Documentation Standards

All interfaces in `domain/interfaces.go` include:
- **Purpose**: What the interface does
- **Implementation**: Where to find concrete classes
- **Created By**: How/where it's instantiated
- **Used By**: Who depends on it
- **Responsibilities**: Key duties

This ensures you can always find:
1. What it does (interface contract)
2. How it works (implementation)
3. Where it's used (dependencies)

---

## 🔄 Typical Workflows

### Adding a New JSON Type Support
```
1. Create processor: infrastructure/execution/my_processor.go
   ├─ Implement: TypeProcessor interface
   └─ Use: LLMProvider, PromptBuilder

2. Register: factory/generator_factory.go:createTypeProcessors()
   └─ Add to processors list

3. Test: Create *_test.go file

Reference: QUICK_REFERENCE.md → "Adding a New Type Processor"
```

### Changing Execution Behavior
```
1. Create strategy: infrastructure/strategies/my_strategy.go
   ├─ Implement: ExecutionStrategy interface
   └─ Logic: Schedule() and Execute() methods

2. Register: factory/generator_factory.go:createStrategy()
   └─ Add case for new mode

3. Config: factory/config.go
   └─ Add mode constant

Reference: QUICK_REFERENCE.md → "Adding a New Execution Strategy"
```

### Understanding a Flow Issue
```
1. Map the flow: ARCHITECTURE_FLOW.md → "Generation Flow"

2. Find components: INTERFACE_IMPLEMENTATION_MAP.md
   └─ Locate implementations involved

3. Add logging: QUICK_REFERENCE.md → "Debugging Tips"
   └─ Key points to instrument

4. Trace execution: Run with logging enabled
```

---

## 🎓 Learning Path

### Week 1: Understanding
- [ ] Read ARCHITECTURE_FLOW.md (Overview, Entry Points)
- [ ] Browse domain/interfaces.go (just read comments)
- [ ] Trace one generation request through ARCHITECTURE_FLOW.md

### Week 2: Exploration
- [ ] Read INTERFACE_IMPLEMENTATION_MAP.md
- [ ] Open each implementation file, read doc comments
- [ ] Run the system, add logging to trace flow

### Week 3: Contributing
- [ ] Pick a simple task (e.g., new processor)
- [ ] Use QUICK_REFERENCE.md templates
- [ ] Write tests following existing patterns

---

## ❓ FAQ

**Q: Where do I find all interfaces?**
A: `domain/interfaces.go` - All interfaces in one place with detailed docs

**Q: Where is Interface X implemented?**
A: Check INTERFACE_IMPLEMENTATION_MAP.md → Find interface → See implementations

**Q: How do I add feature Y?**
A: Check QUICK_REFERENCE.md → "Common Tasks" → Find similar task

**Q: What does component Z do?**
A: Check ARCHITECTURE_FLOW.md → "Component Dependencies" → Find description

**Q: Why is the code organized this way?**
A: Check REFACTORING_PLAN.md → "Problem Statement" and "Solution"

**Q: How do I debug issue W?**
A: ARCHITECTURE_FLOW.md (flow) + QUICK_REFERENCE.md (debugging tips)

---

## 🛠️ Maintenance

### Keeping Docs Updated

When you change code, update:

**Added new interface?**
- Update: domain/interfaces.go (add doc comments)
- Update: INTERFACE_IMPLEMENTATION_MAP.md (add mapping)

**Added new implementation?**
- Update: domain/interfaces.go (add to implementation list in comments)
- Update: INTERFACE_IMPLEMENTATION_MAP.md (add row to table)

**Changed flow?**
- Update: ARCHITECTURE_FLOW.md (update relevant sections)

**Added common pattern?**
- Update: QUICK_REFERENCE.md (add to appropriate section)

---

## 📞 Support

**Can't find what you need?**
1. Check this index
2. Search docs (all are markdown, grep-friendly)
3. Check code comments (especially interfaces.go)
4. Ask the team

**Found an issue?**
- Documentation bug: Update the relevant .md file
- Code issue: Check ARCHITECTURE_FLOW.md to understand impact

---

## 🎉 Quick Wins

Start here for immediate productivity:

1. **Bookmark** domain/interfaces.go (all contracts)
2. **Keep open** QUICK_REFERENCE.md (daily use)
3. **Print** flow diagram from ARCHITECTURE_FLOW.md
4. **Star** key files from INTERFACE_IMPLEMENTATION_MAP.md

Happy coding! 🚀
