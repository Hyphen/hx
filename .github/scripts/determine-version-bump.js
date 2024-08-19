const { execSync } = require('child_process');

function determineVersionBump() {
  // Get the latest release information
  const latestRelease = JSON.parse(process.env.LATEST_RELEASE);
  const latestVersion = latestRelease.version;
  let bumpInfo = latestRelease.bumpInfo;

  console.log(`Latest version: ${latestVersion}`);
  console.log(`Current bump info: ${JSON.stringify(bumpInfo)}`);

  // Get the last 3 commits
  const commits = execSync('git log -n 3 --pretty=format:"%s%n%b"').toString();

  console.log("Analyzing the last 3 commits:");
  console.log(commits);
  console.log("---End of commits---");

  let bumpType = 'patch';

  if (/(\n|^)BREAKING CHANGE:/.test(commits) || /^[^:]+!:/.test(commits)) {
    bumpType = 'major';
  } else if (/^feat(\(.+\))?:/.test(commits)) {
    bumpType = 'minor';
  }

  // Update bump info
  if (bumpType === 'major') {
    bumpInfo = { major: true, minor: true, patch: true };
  } else if (bumpType === 'minor') {
    bumpInfo = { ...bumpInfo, minor: true, patch: true };
  } else {
    bumpInfo = { ...bumpInfo, patch: true };
  }

  console.log(`Determined bump type: ${bumpType}`);
  console.log(`New bump info: ${JSON.stringify(bumpInfo)}`);

  return { bumpType, bumpInfo };
}

// Run the function and output results
const result = determineVersionBump();
console.log(`::set-output name=bump_type::${result.bumpType}`);
console.log(`::set-output name=new_bump_info::${JSON.stringify(result.bumpInfo)}`);
