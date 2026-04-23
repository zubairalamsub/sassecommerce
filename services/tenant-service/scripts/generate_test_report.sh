#!/bin/bash

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

REPORT_DIR="test-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_NAME="test-report-${TIMESTAMP}"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Tenant Service - Test Report Generator${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# Navigate to service directory
cd "$(dirname "$0")/.."

# Create report directory
mkdir -p ${REPORT_DIR}

echo -e "${GREEN}📁 Report directory: ${REPORT_DIR}/${REPORT_NAME}${NC}"
mkdir -p ${REPORT_DIR}/${REPORT_NAME}

# Install gotestsum if not present
if ! command -v gotestsum &> /dev/null; then
    echo -e "${YELLOW}Installing gotestsum...${NC}"
    go install gotest.tools/gotestsum@latest
fi

# Run tests with JSON output
echo -e "\n${BLUE}🧪 Running tests...${NC}"
gotestsum --format testname --jsonfile ${REPORT_DIR}/${REPORT_NAME}/tests.json -- -coverprofile=${REPORT_DIR}/${REPORT_NAME}/coverage.out ./... 2>&1 | tee ${REPORT_DIR}/${REPORT_NAME}/test-output.log

# Generate coverage reports
echo -e "\n${BLUE}📊 Generating coverage reports...${NC}"

# HTML coverage
go tool cover -html=${REPORT_DIR}/${REPORT_NAME}/coverage.out -o ${REPORT_DIR}/${REPORT_NAME}/coverage.html

# Coverage summary
go tool cover -func=${REPORT_DIR}/${REPORT_NAME}/coverage.out > ${REPORT_DIR}/${REPORT_NAME}/coverage-summary.txt

# Get total coverage percentage
COVERAGE=$(go tool cover -func=${REPORT_DIR}/${REPORT_NAME}/coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}Total Coverage: ${COVERAGE}${NC}"

# Generate HTML report
echo -e "\n${BLUE}📝 Generating HTML report...${NC}"

cat > ${REPORT_DIR}/${REPORT_NAME}/index.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tenant Service - Test Report</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: #f5f5f5;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
        }
        .header h1 {
            font-size: 32px;
            margin-bottom: 10px;
        }
        .header p {
            opacity: 0.9;
            font-size: 16px;
        }
        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            padding: 30px;
            background: #f9fafb;
        }
        .metric {
            background: white;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .metric-label {
            color: #6b7280;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }
        .metric-value {
            font-size: 32px;
            font-weight: bold;
            color: #111827;
        }
        .metric.success {
            border-left-color: #10b981;
        }
        .metric.warning {
            border-left-color: #f59e0b;
        }
        .metric.error {
            border-left-color: #ef4444;
        }
        .metric-value.success {
            color: #10b981;
        }
        .metric-value.warning {
            color: #f59e0b;
        }
        .metric-value.error {
            color: #ef4444;
        }
        .section {
            padding: 30px;
            border-top: 1px solid #e5e7eb;
        }
        .section-title {
            font-size: 24px;
            margin-bottom: 20px;
            color: #111827;
        }
        .test-list {
            list-style: none;
        }
        .test-item {
            padding: 15px;
            margin-bottom: 10px;
            background: #f9fafb;
            border-radius: 6px;
            border-left: 4px solid #10b981;
        }
        .test-item.failed {
            border-left-color: #ef4444;
            background: #fef2f2;
        }
        .test-item.skipped {
            border-left-color: #f59e0b;
            background: #fffbeb;
        }
        .test-name {
            font-weight: 600;
            color: #111827;
            margin-bottom: 5px;
        }
        .test-time {
            color: #6b7280;
            font-size: 14px;
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .badge.passed {
            background: #d1fae5;
            color: #065f46;
        }
        .badge.failed {
            background: #fee2e2;
            color: #991b1b;
        }
        .badge.skipped {
            background: #fef3c7;
            color: #92400e;
        }
        .links {
            padding: 20px 30px;
            background: #f9fafb;
            display: flex;
            gap: 15px;
        }
        .btn {
            display: inline-block;
            padding: 10px 20px;
            background: #667eea;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            font-weight: 500;
            transition: background 0.2s;
        }
        .btn:hover {
            background: #5568d3;
        }
        .coverage-bar {
            width: 100%;
            height: 30px;
            background: #e5e7eb;
            border-radius: 15px;
            overflow: hidden;
            margin-top: 10px;
        }
        .coverage-fill {
            height: 100%;
            background: linear-gradient(90deg, #10b981 0%, #059669 100%);
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: 600;
            font-size: 14px;
        }
        .timestamp {
            color: #6b7280;
            font-size: 14px;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🧪 Tenant Service Test Report</h1>
            <p>Comprehensive test execution and coverage analysis</p>
            <p class="timestamp" id="timestamp"></p>
        </div>

        <div class="summary">
            <div class="metric success">
                <div class="metric-label">Total Tests</div>
                <div class="metric-value" id="total-tests">0</div>
            </div>
            <div class="metric success">
                <div class="metric-label">Passed</div>
                <div class="metric-value success" id="passed-tests">0</div>
            </div>
            <div class="metric error">
                <div class="metric-label">Failed</div>
                <div class="metric-value error" id="failed-tests">0</div>
            </div>
            <div class="metric warning">
                <div class="metric-label">Skipped</div>
                <div class="metric-value warning" id="skipped-tests">0</div>
            </div>
            <div class="metric">
                <div class="metric-label">Duration</div>
                <div class="metric-value" id="duration">0s</div>
            </div>
            <div class="metric">
                <div class="metric-label">Coverage</div>
                <div class="metric-value" id="coverage">0%</div>
                <div class="coverage-bar">
                    <div class="coverage-fill" id="coverage-bar" style="width: 0%">
                        <span id="coverage-text">0%</span>
                    </div>
                </div>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">Test Results by Package</h2>
            <div id="test-results"></div>
        </div>

        <div class="section">
            <h2 class="section-title">Coverage by Package</h2>
            <div id="coverage-details"></div>
        </div>

        <div class="links">
            <a href="coverage.html" class="btn" target="_blank">📊 View Detailed Coverage</a>
            <a href="coverage-summary.txt" class="btn" target="_blank">📄 Coverage Summary</a>
            <a href="tests.json" class="btn" target="_blank">📋 Raw JSON Data</a>
        </div>
    </div>

    <script>
        // Load test results
        fetch('tests.json')
            .then(response => response.text())
            .then(data => {
                const lines = data.trim().split('\n');
                const events = lines.map(line => JSON.parse(line));

                // Parse test results
                const tests = {};
                const packages = {};
                let totalDuration = 0;

                events.forEach(event => {
                    if (event.Test) {
                        const key = event.Package + '/' + event.Test;
                        if (!tests[key]) {
                            tests[key] = {
                                package: event.Package,
                                name: event.Test,
                                output: []
                            };
                        }

                        if (event.Action === 'pass' || event.Action === 'fail' || event.Action === 'skip') {
                            tests[key].action = event.Action;
                            tests[key].elapsed = event.Elapsed || 0;
                            totalDuration += tests[key].elapsed;
                        }

                        if (event.Output) {
                            tests[key].output.push(event.Output);
                        }
                    }

                    if (!event.Test && event.Action === 'pass') {
                        packages[event.Package] = 'pass';
                    } else if (!event.Test && event.Action === 'fail') {
                        packages[event.Package] = 'fail';
                    }
                });

                // Calculate statistics
                const testArray = Object.values(tests);
                const passed = testArray.filter(t => t.action === 'pass').length;
                const failed = testArray.filter(t => t.action === 'fail').length;
                const skipped = testArray.filter(t => t.action === 'skip').length;
                const total = testArray.length;

                // Update summary
                document.getElementById('total-tests').textContent = total;
                document.getElementById('passed-tests').textContent = passed;
                document.getElementById('failed-tests').textContent = failed;
                document.getElementById('skipped-tests').textContent = skipped;
                document.getElementById('duration').textContent = totalDuration.toFixed(2) + 's';

                // Group tests by package
                const byPackage = {};
                testArray.forEach(test => {
                    if (!byPackage[test.package]) {
                        byPackage[test.package] = [];
                    }
                    byPackage[test.package].push(test);
                });

                // Render test results
                const resultsDiv = document.getElementById('test-results');
                Object.keys(byPackage).sort().forEach(pkg => {
                    const pkgTests = byPackage[pkg];
                    const pkgDiv = document.createElement('div');
                    pkgDiv.style.marginBottom = '20px';

                    const pkgTitle = document.createElement('h3');
                    pkgTitle.textContent = pkg;
                    pkgTitle.style.marginBottom = '10px';
                    pkgTitle.style.color = '#374151';
                    pkgDiv.appendChild(pkgTitle);

                    const testList = document.createElement('ul');
                    testList.className = 'test-list';

                    pkgTests.forEach(test => {
                        const li = document.createElement('li');
                        li.className = 'test-item';
                        if (test.action === 'fail') li.className += ' failed';
                        if (test.action === 'skip') li.className += ' skipped';

                        const badge = document.createElement('span');
                        badge.className = 'badge ' + test.action + 'ed';
                        badge.textContent = test.action;

                        const name = document.createElement('div');
                        name.className = 'test-name';
                        name.textContent = test.name;

                        const time = document.createElement('div');
                        time.className = 'test-time';
                        time.textContent = (test.elapsed || 0).toFixed(3) + 's';

                        li.appendChild(badge);
                        li.appendChild(name);
                        li.appendChild(time);
                        testList.appendChild(li);
                    });

                    pkgDiv.appendChild(testList);
                    resultsDiv.appendChild(pkgDiv);
                });
            });

        // Load coverage
        fetch('coverage-summary.txt')
            .then(response => response.text())
            .then(data => {
                const lines = data.trim().split('\n');
                const totalLine = lines[lines.length - 1];
                const coverage = totalLine.match(/(\d+\.\d+)%/)[1];

                document.getElementById('coverage').textContent = coverage + '%';
                document.getElementById('coverage-bar').style.width = coverage + '%';
                document.getElementById('coverage-text').textContent = coverage + '%';

                // Parse coverage by package
                const coverageDiv = document.getElementById('coverage-details');
                const packages = lines.slice(0, -1).filter(line => line.includes('.go:'));

                const byPackage = {};
                packages.forEach(line => {
                    const match = line.match(/^(.+?)\/([^\/]+\.go):(\d+):\s+(\S+)\s+(\d+\.\d+)%$/);
                    if (match) {
                        const pkg = match[1] || 'main';
                        if (!byPackage[pkg]) {
                            byPackage[pkg] = { total: 0, count: 0 };
                        }
                        byPackage[pkg].total += parseFloat(match[5]);
                        byPackage[pkg].count++;
                    }
                });

                const pkgCoverage = Object.keys(byPackage).map(pkg => ({
                    name: pkg,
                    coverage: (byPackage[pkg].total / byPackage[pkg].count).toFixed(1)
                })).sort((a, b) => b.coverage - a.coverage);

                pkgCoverage.forEach(pkg => {
                    const div = document.createElement('div');
                    div.style.marginBottom = '10px';
                    div.innerHTML = `
                        <div style="display: flex; justify-content: space-between; margin-bottom: 5px;">
                            <span style="font-weight: 500;">${pkg.name}</span>
                            <span style="font-weight: 600; color: #10b981;">${pkg.coverage}%</span>
                        </div>
                        <div class="coverage-bar" style="height: 20px;">
                            <div class="coverage-fill" style="width: ${pkg.coverage}%; font-size: 12px;"></div>
                        </div>
                    `;
                    coverageDiv.appendChild(div);
                });
            });

        // Set timestamp
        document.getElementById('timestamp').textContent = 'Generated: ' + new Date().toLocaleString();
    </script>
</body>
</html>
EOF

echo -e "${GREEN}✅ HTML report generated: ${REPORT_DIR}/${REPORT_NAME}/index.html${NC}"

# Generate markdown report
cat > ${REPORT_DIR}/${REPORT_NAME}/REPORT.md << EOF
# Test Report - $(date)

## Summary

- **Total Coverage**: ${COVERAGE}
- **Report Location**: \`${REPORT_DIR}/${REPORT_NAME}\`

## Files Generated

1. \`index.html\` - Interactive HTML report
2. \`coverage.html\` - Detailed coverage visualization
3. \`coverage-summary.txt\` - Coverage summary
4. \`tests.json\` - Raw test results (JSON)
5. \`test-output.log\` - Test execution log

## Quick Links

- [View HTML Report](./index.html)
- [View Coverage](./coverage.html)
- [Coverage Summary](./coverage-summary.txt)

## How to View

Open the HTML report in your browser:

\`\`\`bash
open ${REPORT_DIR}/${REPORT_NAME}/index.html
\`\`\`

Or start a local server:

\`\`\`bash
cd ${REPORT_DIR}/${REPORT_NAME}
python3 -m http.server 8000
# Then open http://localhost:8000
\`\`\`

---

Generated by Tenant Service Test Reporter
EOF

# Create latest symlink
rm -f ${REPORT_DIR}/latest
ln -s ${REPORT_NAME} ${REPORT_DIR}/latest

echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}✅ Test Report Generated Successfully!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo -e "\n${BLUE}Report Location:${NC}"
echo -e "  📁 ${REPORT_DIR}/${REPORT_NAME}"
echo -e "  🔗 ${REPORT_DIR}/latest (symlink)\n"
echo -e "${BLUE}Files:${NC}"
echo -e "  📊 index.html - Interactive HTML report"
echo -e "  📈 coverage.html - Detailed coverage"
echo -e "  📄 coverage-summary.txt - Coverage summary"
echo -e "  📋 tests.json - Raw JSON data"
echo -e "  📝 REPORT.md - Markdown report\n"
echo -e "${BLUE}View Report:${NC}"
echo -e "  ${YELLOW}open ${REPORT_DIR}/${REPORT_NAME}/index.html${NC}\n"
echo -e "${BLUE}Or start a server:${NC}"
echo -e "  ${YELLOW}cd ${REPORT_DIR}/${REPORT_NAME} && python3 -m http.server 8000${NC}"
echo -e "  ${YELLOW}Then open: http://localhost:8000${NC}\n"
