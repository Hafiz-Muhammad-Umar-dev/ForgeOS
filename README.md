# DevOS — AI-Native Development Operating System

> **Complete Engineering Specification**
> **Status:** All 10 phases drafted (2026-07-20)
> **Author:** Principal Architecture team
> **Intended use:** 2-year build by a full engineering org. Design-first; implementation now proceeding milestone-by-milestone (Sprint 0 started).

---

## Development

DevOS is implemented as a monorepo. Implementation follows the approved Governance, ADRs,
PRD, and SDD — one milestone and one service at a time.

**Prerequisites:** [mise](https://mise.jdx.dev), Docker (for local infrastructure), and Git.

**Quick start:**

```bash
mise install            # install pinned Go / Node / Python + tooling
./scripts/bootstrap.sh  # create .env, fetch Go deps, install git hooks
task --list             # list available tasks
```

Module import path (matches the git remote): `https://github.com/Hafiz-Muhammad-Umar-dev/ForgeOS`.
See [Repository Layout](#repository-layout) and [Sprint Status](#sprint-status).

---

## What This Is

DevOS is **not an IDE and not a chat app** — it is an operating system for software development. A user expresses intent in natural language over **any** of 9 surfaces (Desktop, Web, Mobile, WhatsApp, Telegram, Discord, Slack, Voice, REST) and DevOS plans, assigns a team of specialized AI agents, executes in isolated workspaces, reviews, tests, commits, deploys, and notifies — autonomously, with humans in the loop at key gates.

---

## The Core Flow

```
Human → Intent → Planner → Agent Team → Workspace → Infrastructure → Deployment → Monitoring
```

---

## Specification Map

| Phase | Document | What it defines |
|-------|----------|-----------------|
| **0. Research** | [00-research/00-research-synthesis](00-research/00-research-synthesis.md) | Landscape, validated patterns, gaps (MetaGPT, CrewAI, LangGraph, A2A, Yjs, OIDC…) |
| **1. Architecture** | [01-architecture/01-system-architecture](01-architecture/01-system-architecture.md) | C4 diagrams, ADRs (001–008), provider/agent/bus/workspace/channel designs |
| **2. Specification** | [02-specification/00-index](02-specification/00-index.md) | API contracts, data models, agent protocol, message bus, channel adapters |
| **3. UI/UX** | [03-uiux/00-index](03-uiux/00-index.md) | Design system, Desktop/Web/Mobile UX, messaging/voice UX |
| **4. Database** | [04-database/01-data-architecture](04-database/01-data-architecture.md) | Migrations, repositories, CQRS projections, sharding, DR |
| **5. Backend** | [05-backend/00-index](05-backend/00-index.md) | Service map, Provider Gateway, Workspace Manager |
| **6. Frontend** | [06-frontend/00-index](06-frontend/00-index.md) | Monorepo, real-time/CRDT sync, Electron/Next/RN tech |
| **7. AI Runtime** | [07-ai-runtime/00-index](07-ai-runtime/00-index.md) | Agent run loop, ACI tool use, context assembly, token optimization |
| **8. Testing** | [08-testing/01-testing-strategy](08-testing/01-testing-strategy.md) | Pyramid, agent behavior testing, load, chaos, security |
| **9. Deployment** | [09-deployment/00-index](09-deployment/00-index.md) | K8s/GitOps, CI/CD, observability, scaling, DR |

---

## Architectural Pillars (Binding)

1. **Channel-agnostic core** — all 9 surfaces emit one `IntentCreated` envelope.
2. **Event-driven bus (NATS JetStream)** — no direct cross-context calls.
3. **Provider abstraction** — `LLMProvider`/`ToolProvider`/`DeployProvider`/`ChannelProvider` ports; no vendor hardcoded.
4. **Microkernel + plugins** — bus, registry, DI are the kernel; agents/providers/channels/tools are plugins.
5. **Workspace isolation** — sealed pods + Secret Proxy (Agent Vault pattern).
6. **CRDT local-first sync** — Yjs across surfaces, offline-capable.
7. **Budget Governor** — multi-agent costs ~3× single-agent; spend is capped.
8. **Human-in-the-loop** — approve/reject gates are first-class DAG nodes.
9. **Observability first** — OTel traces every agent span from day one.
10. **Security by design** — OIDC short-lived creds, least-privilege, audit on every mutation.

---

## Key Research Findings That Shaped This Design

- **Role-based multi-agent is the highest-leverage decision** (ChatDev ablation: removing roles dropped executability 88%→58%).
- **Structured artifacts > free-form chat** for inter-agent contracts (MetaGPT).
- **Provider abstraction is proven and necessary** (every major framework ships it).
- **Sandboxed execution is table stakes** (SWE-agent, OpenHands, ChatDev).
- **CRDT + local-first** is the proven multi-device sync model (Yjs, Liveblocks, PartyKit).
- **OIDC + Agent Vault** is the secure secret pattern for agents (GitHub OIDC, Infisical).
- **Token cost** must be governed (multi-agent ~3× single-agent).

---

## How to Use This Document

1. Start with **Phase 0** (why) → **Phase 1** (what/shape) → then any deep-dive phase.
2. Each phase's `00-index.md` links its sub-documents.
3. Cross-references use `(Phase X.Y)` notation.
4. ADRs in Phase 1 are the binding decisions; later phases implement them.

---

## Open Questions for Build Phase

1. Bus technology finalization (NATS vs Kafka) — ADR-001 recommends NATS.
2. Workspace runtime: container vs Firecracker microVM (security tiering).
3. Voice provider selection (Realtime API vs self-hosted ASR/TTS).
4. Exact agent taxonomy v1 (17 proposed; prioritize Planner/Frontend/Backend/DB/QA/Reviewer/Deploy).
5. Self-host vs managed for PG/Redis/NATS at launch.

---

*End of DevOS specification. This is a living document — update ADRs and phases as the build progresses.*

---

## Repository Layout

```
ai-native-os/
├── apps/         # deployable binaries (gateway implemented in Sprint 0; others later)
├── services/     # backend microservices (future milestones)
├── plugins/      # swappable agents / providers / channels / tools (future)
├── core/         # microkernel: bus, registry, di, cqrs, domain, scheduler
├── packages/     # shared libs: contracts + Go foundation (log, otel, config, secrets)
├── infra/        # IaC: docker / k8s / helm / terraform
├── tests/        # unit | integration | e2e | load | chaos
├── scripts/      # dev / build automation (bootstrap, commit-msg check)
├── governance/   # constitution, ADRs, standards, RFCs
├── specs/        # engineering specifications (phases 0–9)
├── sdd/          # service design documents
├── planning/     # build order, milestones, dependency graph
├── product/      # PRD
└── research/     # landscape research
```

Go workspace members: `core`, `packages/go`, `apps/gateway`.

## Sprint Status

| Milestone | Status |
|-----------|--------|
| **Sprint 0 — Monorepo Foundation** | ✅ Component 1 complete (repo structure, go.work, modules, tooling, lint/format, README, bootstrap) |
| Sprint 0 — Dev Environment / Docker / Dev Containers | ⏳ pending |
| Sprint 0 — CI/CD | ⏳ pending |
| Sprint 0 — Logging / OTel / Config / Secrets | ⏳ pending |
| Sprint 0 — NATS Event Bus | ⏳ pending |
| Sprint 0 — Kernel skeleton | ⏳ pending |
| Sprint 0 — API Gateway skeleton | ⏳ pending |

See `planning/03-build-order.md` and `planning/01-product-milestones.md` for the full sequence.

## License

Licensed under the [Apache License 2.0](LICENSE). The definitive license is pending the
license ADR (governance/03-adr.md); Apache-2.0 is applied provisionally.
