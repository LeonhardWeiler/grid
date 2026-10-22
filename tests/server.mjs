import { spawn, spawnSync } from "node:child_process"
import { mkdtempSync, rmSync } from "node:fs"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"

const work = mkdtempSync(join(tmpdir(), "grid-browser-"))
for (const [command, args] of [
  ["npm", ["run", "build"]],
  ["go", ["build", "-o", join(work, "grid"), "."]]
]) {
  const result = spawnSync(command, args, { stdio: "inherit" })
  if (result.status !== 0) {
    rmSync(work, { recursive: true, force: true })
    process.exit(1)
  }
}
const child = spawn(join(work, "grid"), [], {
  stdio: "inherit",
  env: {
    ...process.env,
    STATE_DIRECTORY: work,
    LISTEN_ADDR: "127.0.0.1:5410",
    STATIC_DIR: resolve("public/dist"),
    ADMIN_ADDR: ""
  }
})
for (const signal of ["SIGINT", "SIGTERM"])
  process.on(signal, () => child.kill("SIGTERM"))
child.on("exit", (code) => {
  rmSync(work, { recursive: true, force: true })
  process.exit(code ?? 1)
})
