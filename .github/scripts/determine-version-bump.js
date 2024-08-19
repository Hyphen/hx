const { execSync } = require('child_process');

function determineVersionBump() {
  const latestReleaseString = process.env.LATEST_RELEASE;
  const latestRelease = JSON.parse(latestReleaseString.replace(/^"(.*)"$/, '$1').replace(/\\"/g, '"'));
  const latestVersion = latestRelease.version;
  let bumpInfo = latestRelease.bumpInfo;

  console.error(`Debug: Latest version: ${latestVersion}`);
  console.error(`Debug: Current bump info: ${JSON.stringify(bumpInfo)}`);

  const commits = execSync('git log -n 3 --pretty=format:"%s%n%b"').toString();

  console.error("Debug: Analyzing the last 3 commits:");
  console.error(commits);
  console.error("---End of commits---");

  let bumpType = 'patch';

  if (/(\n|^)BREAKING CHANGE:/.test(commits) || /^[^:]+!:/.test(commits)) {
    bumpType = 'major';
  } else if (/^feat(\(.+\))?:/m.test(commits)) {  
    bumpType = 'minor';
  }

  if (bumpType === 'major') {
    bumpInfo = { major: true, minor: true, patch: true };
  } else if (bumpType === 'minor') {
    bumpInfo = { ...bumpInfo, minor: true, patch: true };
  } else {
    bumpInfo = { ...bumpInfo, patch: true };
  }

  console.error(`Debug: Determined bump type: ${bumpType}`);
  console.error(`Debug: New bump info: ${JSON.stringify(bumpInfo)}`);

  return { bumpType, bumpInfo };
}

const result = determineVersionBump();
console.log(`bump_type=${result.bumpType}`);
console.log(`new_bump_info=${JSON.stringify(result.bumpInfo)}`);
