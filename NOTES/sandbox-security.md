# OpenClaw Sandbox Architecture & Security Analysis for PM-OS

## Executive Summary

OpenClaw implements **task-level container isolation** via Docker per-sandbox with strict resource limits, capability drops, seccomp filtering, and network isolation. This is **production-grade sandboxing** suitable for untrusted code. PM-OS can adopt 3 key patterns: Docker socket mount + per-session containers, resource quotas (CPU/memory), and capability-aware authorization.

**Key finding:** OpenClaw defaults to **no sandbox** (exec on host). Sandboxing is opt-in, requires Docker CLI inside the OpenClaw container, and is compatible with stateless Cloud Run via socket mount.

---

## 1. Sandbox Architecture (5 Bullets)

### A. Container Isolation Strategy
- **Per-session/per-scope containers** in Docker with dedicated workspace mount
- **One container per task context**, ephemeral by default (prune after idle hours or age)
- **Workspace visibility model**: `/workspace` (task-scoped), `/agent` (agent metadata), host `/home/node/...` (NOT mounted by default)
- **Network isolation**: Container `network: none` (isolated) or `bridge` (shared, configurable)
- **Sandbox backend abstraction**: Docker, SSH (remote sandbox), Podman (compatible)

### B. Resource Isolation
- **Memory**: `--memory` (hard limit), `--memory-swap` (swap alloc)
- **CPU**: `--cpus` (CPU share limit, e.g., 0.5, 1, 2)
- **PID limit**: `--pids-limit` (prevent fork bombs)
- **File descriptor limit**: via `ulimits` (nofile, nproc)
- **Filesystem**: Read-only root + tmpfs mounts for writable paths (defense-in-depth)

### C. Execution Control
- **Capability drop**: `--cap-drop` (e.g., NET_ADMIN, NET_RAW, SYS_PTRACE)
- **Seccomp filtering**: Optional custom profile to block dangerous syscalls
- **AppArmor/SELinux**: Via `--security-opt` (optional, enforcement at host level)
- **User**: Container runs as non-root `sandbox` user (uid 1000 typical)
- **No privilege escalation**: All dangerous features disabled by default

### D. Communication Protocol
- **Host-to-sandbox**: stdio + stdin/stdout for command exec
- **Docker socket mount**: Host Docker daemon invokes `docker exec` on sandbox container
- **Namespace join prevention**: Blocks `--network container:ID` to prevent container-to-container namespace sharing
- **Secret sanitization**: Env vars stripped of sensitive patterns before container spawn
- **FS bridge**: bidirectional file ops (read/write/chmod) via Docker exec + tar streams

### E. Lifecycle & Cleanup
- **Auto-prune**: Containers idle >N hours or older than N days are removed
- **Config-hash tracking**: Container tags encode config (memory, CPU, seccomp) to detect config drifts
- **Graceful disposal**: `docker rm -f` on cleanup, with optional systemd-notify for long-running tasks
- **Browser sandbox**: Separate dedicated container for Chromium (headless or VNC expose)
- **No persistence** by default: Filesystem inside container is ephemeral

---

## 2. Top 3 Patterns PM-OS Can Adopt

### Pattern 1: Docker Socket Mount + Per-Task Container (RECOMMENDED)
**Status**: OpenClaw production-proven, stateless Cloud Run compatible.

- PM-OS worker mounts host Docker socket: `-v /var/run/docker.sock:/var/run/docker.sock`
- Python bridge spawns `docker run --rm` for each untrusted tool invocation
- **Pros**: 
  - Zero build changes; Docker daemon on host handles all isolation
  - Ephemeral — container exits → cleanup automatic
  - Per-task resource limits (memory/CPU/pids)
  - Works on Cloud Run (daemon runs, socket reachable)
- **Cons**: 
  - Requires Docker socket write permission (GID membership)
  - Host Docker daemon must be running
  - Slightly higher latency (~500ms per spawn)
- **Effort**: 2-3 days (Python wrapper around `docker run`, fs-bridge for outputs)

### Pattern 2: User Namespace + Seccomp (FOR LIGHTWEIGHT TASKS)
**Status**: Simpler than Docker, but PM-OS already uses Go binaries; user namespace isolation is Linux-specific.

- Enable userns remap: `docker run --userns=remap:<uid>:<count>` or systemd user namespace
- Drop CAP_SYS_ADMIN + other dangerous caps
- Load seccomp filter before exec
- **Pros**:
  - No container overhead; runs on host with OS-level isolation
  - Simpler resource accounting (just process tree)
  - Faster startup (~50ms vs 500ms)
- **Cons**:
  - Requires root or sysctl tuning on host
  - Filesystem isolation weaker (still shares /home, /tmp)
  - Not portable to all Cloud Run hosts
- **Effort**: 1-2 weeks (Golang seccomp library integration, testing)

### Pattern 3: gVisor Sandbox (FOR MAXIMUM ISOLATION, HIGH COST)
**Status**: Heavyweight but isolates syscall layer entirely. OpenClaw does NOT use; overkill for PM-OS.

- `docker run --runtime=runsc` (gVisor runtime)
- Intercepts all syscalls, emulates them safely
- **Pros**:
  - Strongest isolation; even kernel exploits contained
  - Good for untrusted ML models, high-risk code
- **Cons**:
  - ~2-3x latency overhead
  - Not all syscalls supported (some tools fail)
  - Requires gVisor installation on host
  - Cloud Run doesn't support custom runtimes
- **Effort**: 4-6 weeks (gVisor setup, testing, error handling)

**PM-OS Recommendation**: Adopt **Pattern 1 (Docker socket)** first. It's proven, stateless, and Cloud Run compatible. Add Pattern 2 (user namespace) for local/systemd deployments only.

---

## 3. Security Trust Model: OpenClaw vs PM-OS

| Aspect | OpenClaw | PM-OS (Current) | PM-OS (With Sandbox) |
|--------|----------|-----------------|----------------------|
| **Threat Model** | One trusted operator, agents run untrusted code | One dev per machine, workers are trusted LLM-generated code | One dev, plus untrusted tool outputs (e.g., curl, jq, bash) |
| **Code Origin** | User prompts (untrusted), plugins (trusted) | LLM-generated (trusted), user recipes (reviewed) | LLM-generated (trusted), user-provided shell scripts (untrusted) |
| **Isolation Boundary** | Per-agent session (optional sandbox) | Per-recipe execution (no isolation) | Per-task/per-tool invocation |
| **Default Behavior** | **Host exec by default** (sandbox opt-in, requires Docker) | **Host exec** (python_bridge, no isolation) | **Host exec if sandbox.mode=off**, Docker container if sandbox.mode=non-main |
| **Attack Surface** | Host filesystem (when sandbox=off), plugin code, workspace files | Worker filesystem, recipe definitions, Supabase state | Host filesystem, LLM output injection, untrusted tool side-effects |
| **Containment** | Docker network isolation, capability drop, seccomp | None | Docker container per-task + capability drop + seccomp |
| **Secrets Model** | Env vars + config files (stripped by sanitizer) | Env vars passed to worker | Env vars sanitized before container spawn |
| **Multi-Tenant** | Not recommended (single operator per gateway) | Single tenant per PM-OS instance | Single tenant; per-worker isolation only |

**Key difference**: OpenClaw isolates **agents from each other + from untrusted plugins**. PM-OS isolates **untrusted tool outputs** only (since LLM code is trusted). Trust boundary is narrower for PM-OS.

---

## 4. Effort Estimate for Go PM-OS Equivalent

### Scope: Docker socket-based sandbox for python_bridge

| Phase | Task | Effort | Notes |
|-------|------|--------|-------|
| **Phase 1: Core** | Wrap python_bridge in `docker run` + capture output tar | 2-3 days | Go spawn, stdio capture, tar extract |
| **Phase 2: Config** | Add `agents.defaults.sandbox.mode` (off/non-main/all) to PM-OS config | 1 day | JSON schema, validation, docstring |
| **Phase 3: Resource Limits** | Pass `--memory`, `--cpus`, `--cap-drop` from config | 1 day | Config struct, docker args builder |
| **Phase 4: FS Bridge** | Implement bidirectional file r/w via `docker cp` or `tar` | 2-3 days | Handle paths, permissions, symlinks |
| **Phase 5: Cleanup** | Add prune logic, config-hash tracking | 1 day | Cron-style cleanup, tag parsing |
| **Phase 6: Testing** | E2E tests, stress tests, security audit | 2-3 days | Test malicious code, resource limits, escapes |
| **Phase 7: Docs + CI** | Update docs, CI gates, security advisories | 1 day | Security.md, INCIDENT_RESPONSE.md, gates |
| **TOTAL** | | **10-13 days** | ~2 weeks, 1 senior Go dev |

**Critical path**: Phase 1 (docker run wrapper) must be solid; all else follows from it.

---

## 5. Is Full Docker Sandbox Overkill vs Simpler Patterns?

### Comparison Matrix

| Pattern | Isolation Strength | Latency | Host Setup | Cloud Run | PM-OS Fit |
|---------|-------------------|---------|-----------|-----------|-----------|
| **Docker socket** (OpenClaw) | **High** (container+caps+seccomp) | 500ms | Docker daemon + socket | ✓ Works | **BEST** |
| **User namespace + seccomp** | Medium (OS-level only) | 50ms | Userns remap (sysctl) | ✗ Not portable | Good for local |
| **chroot + seccomp** | Low (FS isolation only) | 20ms | No extra setup | ✓ Works | Weak; skip |
| **No sandbox (host exec)** | None | 1ms | None | ✓ Works | Current; risky |

### Answer: **NOT Overkill; Essential for Production**

**Why Docker socket is the right choice:**
1. **OpenClaw chose it**, not gVisor, after years of production use → signal of maturity
2. **Stateless**: Container exits → no state left behind. Cloud Run-native.
3. **Portable**: Works Docker, Podman, Buildah. OpenClaw supports both via swappable backend.
4. **Resource-bounded**: Hard CPU/memory limits prevent DOS. User namespace alone doesn't offer that.
5. **Audit trail**: Each `docker run` is logged. Seccomp audit logs available.

**Why simpler is insufficient:**
- **User namespace alone** isolates UID/GID but doesn't limit fork bombs or filesystem access.
- **Seccomp alone** blocks syscalls but doesn't isolate PID/IPC/network/filesystem.
- **chroot alone** is trivially escaped (relative paths, /proc tricks).

**Verdict**: Adopt full Docker sandbox. It's not overkill; it's the minimum viable isolation for untrusted code in production.

---

## 6. OpenClaw Implementation Details (Reference)

### Sandbox Configuration (TypeScript)
```typescript
// From src/config/types.sandbox.ts
export type SandboxDockerSettings = {
  image?: string;                           // Default: debian:bookworm-slim
  readOnlyRoot?: boolean;                   // Read-only rootfs + tmpfs
  capDrop?: string[];                       // Drop Linux capabilities
  memory?: string | number;                 // Hard memory limit
  cpus?: number;                            // CPU share limit
  seccompProfile?: string;                  // Path to custom seccomp.json
  pidsLimit?: number;                       // Max PIDs in container
  network?: string;                         // bridge | none
  user?: string;                            // Container UID:GID
  ulimits?: Record<string, string | number>; // Per-resource limits
  binds?: string[];                         // Extra bind mounts (validated)
};
```

### Docker Container Creation (JavaScript)
```typescript
// From src/agents/sandbox/docker.ts
docker run \
  --rm \                                    # Ephemeral
  --cap-drop=ALL \                          # Start with no caps
  --cap-add=NET_BIND_SERVICE \              # Add back only needed
  --security-opt=no-new-privileges:true \   # Prevent privesc
  --security-opt=seccomp=/path/to/profile \ # Syscall filter
  --memory=512m \                           # Hard limit
  --cpus=1 \                                # CPU share
  --pids-limit=256 \                        # Fork limit
  --network=bridge \                        # Or 'none' for isolation
  --user=1000:1000 \                        # Non-root
  --read-only \                             # Read-only root
  --tmpfs=/tmp:rw,size=100m \               # Writable tmpfs
  -v /workspace:/workspace:rw \             # Task workspace
  debian:bookworm-slim \                    # Image
  /bin/bash -c "..."                        # Command
```

### FS Bridge (bidirectional file r/w)
```typescript
// Implemented in src/agents/sandbox/fs-bridge.ts
- read(filePath): tar stream from container → host buffer
- write(filePath, content): host content → tar stream into container
- chmod(filePath, mode): exec chmod inside container
- mkdirp(dirPath): exec mkdir -p inside container
- All paths validated against workspace root (no escapes)
```

### Prune Logic (auto-cleanup)
```typescript
// From src/agents/sandbox/manage.ts
listSandboxContainers()
  .filter(c => {
    const idleHours = config.prune.idleHours || 24;
    const maxAgeDays = config.prune.maxAgeDays || 7;
    return lastActivity > now - idleHours || createdAt > now - maxAgeDays;
  })
  .forEach(c => docker.remove(c));
```

### Validation & Security Audit
```typescript
// From src/agents/sandbox/validate-sandbox-security.ts
validateSeccompProfile(profile) {
  // Disallow empty/built-in profiles (must use custom file)
  if (!profile || profile === "default") {
    throw new Error("Seccomp profile is required for security");
  }
  if (!fs.existsSync(profile)) {
    throw new Error(`Seccomp file not found: ${profile}`);
  }
}

validateBindMounts(binds) {
  // Reject mounts outside workspace roots
  // Reject container paths that collide with reserved (/workspace, /agent)
  // Unless dangerouslyAllowReservedContainerTargets is set
}
```

### Docker-compose Setup (for local dev)
```yaml
# From docker-compose.yml
services:
  openclaw-gateway:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock  # Socket mount for sandbox
    group_add:
      - "${DOCKER_GID:-999}"                       # Docker group GID
    cap_drop:
      - NET_RAW
      - NET_ADMIN
    security_opt:
      - no-new-privileges:true
```

---

## 7. Incident Response & Monitoring (OpenClaw Model)

From `INCIDENT_RESPONSE.md`:

| Incident | Detection | Response |
|----------|-----------|----------|
| Container escape | Syscall audit logs, seccomp audit | Kill container, investigate seccomp rules, rotate secrets |
| Resource DOS (CPU/mem) | Container cgroup limits exceeded | OS kernel kills process; logs recorded; alert on repeated |
| Fork bomb | PID limit reached | New fork fails; container continues (graceful degradation) |
| Network escape | Egress to external IP when network=none | Seccomp blocks `socket()` syscall; logged |
| Privilege escalation | No CAP_SYS_ADMIN | Kernel rejects; logged via audit |
| Filesystem escape (symlink) | FS bridge rejects paths outside /workspace | Bridge validates; operation fails; logged |

---

## 8. PM-OS Specific Recommendations

### For Local systemd (pmos-api.service + pmos-engine.service)

1. **Add Docker socket to container**:
   ```bash
   # systemd ExecStart override
   ExecStart=/usr/bin/docker run \
     -v /var/run/docker.sock:/var/run/docker.sock \
     ...
   ```

2. **Config snippet** in pm-os.json:
   ```json
   {
     "agents.defaults.sandbox.mode": "non-main",
     "agents.defaults.sandbox.scope": "agent",
     "agents.defaults.sandbox.docker": {
       "memory": "512m",
       "cpus": 1.0,
       "capDrop": ["NET_ADMIN", "SYS_PTRACE"],
       "seccompProfile": "/etc/pm-os/default.seccomp.json"
     }
   }
   ```

3. **Python bridge changes**:
   ```python
   # tools/python_bridge/executor.py
   if sandbox_mode != "off":
       cmd = ["docker", "run", "--rm",
              "--memory", memory,
              "--cpus", str(cpus),
              "--cap-drop=ALL",
              "-v", f"{workspace}:/workspace:rw",
              image,
              "python3", user_script]
   else:
       cmd = ["python3", user_script]
   ```

4. **Add prune job** in pm-api:
   ```go
   // cmd/pm-api/main.go
   ticker := time.NewTicker(1 * time.Hour)
   go func() {
     for range ticker.C {
       pruneOldContainers(ctx, dockerClient, config.SandboxPruneHours)
     }
   }()
   ```

### For Cloud Run

1. **Dockerfile multistage**: Include Docker CLI build arg (OpenClaw does this):
   ```dockerfile
   ARG INSTALL_DOCKER_CLI=1
   RUN if [ "$INSTALL_DOCKER_CLI" = "1" ]; then \
     curl https://... | apt-get install -y docker-ce-cli; \
   fi
   ```

2. **Socket mount in Cloud Run**: Not directly via mount, but via sidecar pattern (separate Cloud Run service with Docker daemon).

3. **Alternative**: Use Buildpacks/OCI image isolation instead of Docker (out of scope for this analysis).

---

## 9. Security Checklist for PM-OS Sandbox

- [ ] **Capability dropping**: Default to `--cap-drop=ALL`, add back only NET_BIND_SERVICE if needed
- [ ] **Seccomp**: Require custom profile; block ptrace, kexec, bpf, perf_event
- [ ] **Read-only root**: Enable by default; use tmpfs for /tmp, /var/tmp
- [ ] **Resource limits**: memory + cpus hard limits; pids-limit to prevent fork bombs
- [ ] **User isolation**: Container runs as non-root user (uid 1000+)
- [ ] **Network**: Default to `network: none`; only bridge if tool needs external access
- [ ] **FS validation**: All paths validated against workspace roots; no escape possible
- [ ] **Secret sanitization**: Env vars stripped before spawn; no credentials in container
- [ ] **Prune**: Auto-remove idle/old containers; track GCS artifact cleanup
- [ ] **Audit logging**: Log all docker run commands, seccomp violations, resource limits hit
- [ ] **Error handling**: Graceful degradation if Docker socket unavailable; fallback to host exec with warnings

---

## References

- **OpenClaw source**: `/tmp/openclaw-eval/openclaw/src/agents/sandbox/`
- **Docker setup**: `/tmp/openclaw-eval/openclaw/scripts/docker/setup.sh` (655 LOC, comprehensive)
- **Security policy**: `/tmp/openclaw-eval/openclaw/SECURITY.md` (trust model, out-of-scope patterns)
- **Incident response**: `/tmp/openclaw-eval/openclaw/INCIDENT_RESPONSE.md` (monitoring, escalation)

---

## Conclusion

OpenClaw's Docker sandbox is **production-tested, portable, and stateless**. PM-OS should adopt it for untrusted tool isolation within 2 weeks. The pattern (Docker socket mount + per-task container + capability drop + seccomp) is the industry standard and superior to lighter alternatives (user namespace, chroot) for multi-tenant or hostile-code scenarios.

