// Updates the hard-coded version string in the CLI before a release.
// Usage: node scripts/replace.mjs <version>
import { readFileSync, writeFileSync } from 'node:fs'

const version = process.argv[2]
if (!version) {
    console.error('Usage: node scripts/replace.mjs <version>')
    process.exit(1)
}

const file = 'cmd/main.go'
const content = readFileSync(file, 'utf8')
const updated = content.replace(/(Version:\s*)"[^"]*"/, `$1"${version}"`)
writeFileSync(file, updated)

console.log(`Updated ${file} to version ${version}`)
