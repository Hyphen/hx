const { execSync } = require('child_process');

function updateVersion() {
  const latestReleaseString = process.env.LATEST_RELEASE;
  const latestRelease = JSON.parse(latestReleaseString.replace(/^"(.*)"$/, '$1').replace(/\\"/g, '"'));
  const version = latestRelease.version;
  const bumpType = process.env.BUMP_TYPE;
  const currentBumpInfo = latestRelease.bumpInfo;

  console.error("Debug Information");
  console.error(`Current version: ${version}`);
  console.error(`Bump type: ${bumpType}`);
  console.error(`Current bump info: ${JSON.stringify(currentBumpInfo)}`);

  const versionRegex = /^(\d+)\.(\d+)\.(\d+)(?:-rc\.(\d+))?$/;
  const match = version.match(versionRegex);

  if (!match) {
    console.error("Version format is invalid.");
    process.exit(1);
  }

  let [, major, minor, patch, rcNumber] = match;
  major = parseInt(major);
  minor = parseInt(minor);
  patch = parseInt(patch);
  rcNumber = rcNumber ? parseInt(rcNumber) : null;

  const baseVersion = `${major}.${minor}.${patch}`;

  console.error(`Base version: ${baseVersion}`);
  console.error(`RC number: ${rcNumber}`);

  let newBumpInfo, newVersion;

  if (bumpType === "major") {
    newBumpInfo = { major: true, minor: true, patch: true };
    newVersion = `${major + 1}.0.0-rc.1`;
    console.error(`New bump info for major: ${JSON.stringify(newBumpInfo)}`);
  } else if (bumpType === "minor") {
    newBumpInfo = { ...currentBumpInfo, minor: true, patch: true };
    console.error(`New bump info for minor: ${JSON.stringify(newBumpInfo)}`);
    if (!currentBumpInfo.minor) {
      newVersion = `${major}.${minor + 1}.0-rc.1`;
      console.error(`New version for minor bump when minor was false: ${newVersion}`);
    } else {
      if (!rcNumber || isNaN(rcNumber)) {
        console.error("RC number is empty or invalid, setting to 1");
        rcNumber = 1;
      } else {
        console.error(`Current RC number: ${rcNumber}`);
        rcNumber++;
      }
      newVersion = `${baseVersion}-rc.${rcNumber}`;
      console.error(`New version for minor bump when minor was true: ${newVersion}`);
    }
  } else {
    newBumpInfo = { ...currentBumpInfo, patch: true };
    console.error(`New bump info for patch: ${JSON.stringify(newBumpInfo)}`);
    if (!currentBumpInfo.patch) {
      newVersion = `${major}.${minor}.${patch + 1}-rc.1`;
      console.error(`New version for patch bump when patch was false: ${newVersion}`);
    } else {
      if (!rcNumber || isNaN(rcNumber)) {
        console.error("RC number is empty or invalid, setting to 1");
        rcNumber = 1;
      } else {
        console.error(`Current RC number: ${rcNumber}`);
        rcNumber++;
      }
      newVersion = `${baseVersion}-rc.${rcNumber}`;
      console.error(`New version for patch bump when patch was true: ${newVersion}`);
    }
  }

  console.error(`Final new version: ${newVersion}`);
  console.error(`Final new bump info: ${JSON.stringify(newBumpInfo)}`);

  return { newVersion, newBumpInfo };
}

const result = updateVersion();
console.log(`new_version=${result.newVersion}`);
console.log(`bump_info=${JSON.stringify(result.newBumpInfo)}`);
