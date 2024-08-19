const core = require('@actions/core');
const { execSync } = require('child_process');

try {
  const latestRelease = process.env.latest_release.replace(/^"(.*)"$/, '$1').replace(/\\"/g, '"');
  const latestVersion = JSON.parse(latestRelease).version;
  const bumpInfo = JSON.parse(latestRelease).bumpInfo;

  console.log("Latest version:", latestVersion);
  console.log("Current bump info:", bumpInfo);

  console.log("Analyzing the last 3 commits:");
  const commits = execSync('git log -n 3 --pretty=format:"%s%n%b"').toString();
  console.log("Commits to analyze:");
  console.log(commits);
  console.log("---End of commits---");

  let bumpType;
  if (/(\n|^)BREAKING CHANGE:/.test(commits) || /^[^:]+!/.test(commits)) {
    bumpType = "major";
  } else if (/^feat(\(.+\))?:/.test(commits)) {
    bumpType = "minor";
  } else {
    bumpType = "patch";
  }

  const newBumpInfo = {
    ...bumpInfo,
    major: bumpType === "major" || bumpInfo.major,
    minor: bumpType === "major" || bumpType === "minor" || bumpInfo.minor,
    patch: bumpType !== "patch" ? true : bumpInfo.patch,
  };

  console.log("Determined bump type:", bumpType);
  core.setOutput('bump_type', bumpType);
  core.setOutput('new_bump_info', JSON.stringify(newBumpInfo));

} catch (error) {
  core.setFailed(error.message);
}
