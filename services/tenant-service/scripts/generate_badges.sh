#!/bin/bash

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

BADGE_DIR="badges"

echo -e "${BLUE}Generating Test Badges...${NC}"

# Navigate to service directory
cd "$(dirname "$0")/.."

# Create badge directory
mkdir -p ${BADGE_DIR}

# Run tests and get coverage
go test -coverprofile=coverage.out ./... > /dev/null 2>&1

# Get coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

# Determine coverage color
if (( $(echo "$COVERAGE >= 80" | bc -l) )); then
    COVERAGE_COLOR="brightgreen"
elif (( $(echo "$COVERAGE >= 60" | bc -l) )); then
    COVERAGE_COLOR="yellow"
else
    COVERAGE_COLOR="red"
fi

# Count tests
TOTAL_TESTS=$(go test -v ./... 2>&1 | grep -c "RUN")
PASSED_TESTS=$(go test -v ./... 2>&1 | grep -c "PASS: Test")

# Generate coverage badge JSON
cat > ${BADGE_DIR}/coverage.json << EOF
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "${COVERAGE}%",
  "color": "${COVERAGE_COLOR}"
}
EOF

# Generate tests badge JSON
cat > ${BADGE_DIR}/tests.json << EOF
{
  "schemaVersion": 1,
  "label": "tests",
  "message": "${TOTAL_TESTS} tests",
  "color": "blue"
}
EOF

# Generate passing badge JSON
cat > ${BADGE_DIR}/passing.json << EOF
{
  "schemaVersion": 1,
  "label": "build",
  "message": "passing",
  "color": "brightgreen"
}
EOF

# Generate SVG badges using shields.io endpoint format
cat > ${BADGE_DIR}/coverage.svg << EOF
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="108" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="a">
    <rect width="108" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#a)">
    <path fill="#555" d="M0 0h61v20H0z"/>
    <path fill="#${COVERAGE_COLOR}" d="M61 0h47v20H61z"/>
    <path fill="url(#b)" d="M0 0h108v20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110">
    <text x="315" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="510">coverage</text>
    <text x="315" y="140" transform="scale(.1)" textLength="510">coverage</text>
    <text x="835" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="370">${COVERAGE}%</text>
    <text x="835" y="140" transform="scale(.1)" textLength="370">${COVERAGE}%</text>
  </g>
</svg>
EOF

cat > ${BADGE_DIR}/tests.svg << EOF
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="88" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="a">
    <rect width="88" height="20" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#a)">
    <path fill="#555" d="M0 0h37v20H0z"/>
    <path fill="#007ec6" d="M37 0h51v20H37z"/>
    <path fill="url(#b)" d="M0 0h88v20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110">
    <text x="195" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="270">tests</text>
    <text x="195" y="140" transform="scale(.1)" textLength="270">tests</text>
    <text x="615" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="410">${TOTAL_TESTS}</text>
    <text x="615" y="140" transform="scale(.1)" textLength="410">${TOTAL_TESTS}</text>
  </g>
</svg>
EOF

# Generate README badge markdown
cat > ${BADGE_DIR}/README_BADGES.md << EOF
# Test Badges

Add these to your README.md:

## Using JSON endpoint (shields.io)

\`\`\`markdown
![Coverage](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/YOUR_USERNAME/YOUR_REPO/main/services/tenant-service/badges/coverage.json)
![Tests](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/YOUR_USERNAME/YOUR_REPO/main/services/tenant-service/badges/tests.json)
![Build](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/YOUR_USERNAME/YOUR_REPO/main/services/tenant-service/badges/passing.json)
\`\`\`

## Using local SVG files

\`\`\`markdown
![Coverage](./badges/coverage.svg)
![Tests](./badges/tests.svg)
\`\`\`

## Using GitHub Actions badge

\`\`\`markdown
![Tests](https://github.com/YOUR_USERNAME/YOUR_REPO/actions/workflows/tenant-service-tests.yml/badge.svg)
\`\`\`

## Current Stats

- **Coverage**: ${COVERAGE}%
- **Total Tests**: ${TOTAL_TESTS}
- **Status**: Passing ✅

---

*Badges auto-generated on $(date)*
EOF

echo -e "${GREEN}✅ Badges generated in ${BADGE_DIR}/${NC}"
echo -e "${BLUE}Coverage: ${COVERAGE}%${NC}"
echo -e "${BLUE}Total Tests: ${TOTAL_TESTS}${NC}"
echo -e "\nGenerated files:"
echo "  - ${BADGE_DIR}/coverage.json"
echo "  - ${BADGE_DIR}/coverage.svg"
echo "  - ${BADGE_DIR}/tests.json"
echo "  - ${BADGE_DIR}/tests.svg"
echo "  - ${BADGE_DIR}/passing.json"
echo "  - ${BADGE_DIR}/README_BADGES.md"
