const core = require('@actions/core');
const github = require('@actions/github');
const { execSync } = require('child_process');

function extractLatestRelease(latestRelease) {
  return latestRelease.replace(/^"(.*)"$/, '$1').replace(/\\"/g, '"');
}

function handleMajorBump(currentBumpInfo, major) {
  const newBumpInfo = JSON.stringify({ ...currentBumpInfo, major: true, minor: true, patch: true });
  const newVersion = `${parseInt(major) + 1}.0.0-rc.1`;
  console.log(`New bump info for major: ${newBumpInfo}`);
  return { newBumpInfo, newVersion };
}

function handleMinorBump(currentBumpInfo, major, minor, baseVersion, rcNumber) {
  let newBumpInfo = JSON.stringify({ ...currentBumpInfo, minor: true, patch: true });
  console.log(`New bump info for minor: ${newBumpInfo}`);
  let newVersion;

  if (!currentBumpInfo.minor) {
    newVersion = `${major}.${parseInt(minor) + 1}.0-rc.1`;
    console.log(`New version for minor bump when minor was false: ${newVersion}`);
  } else {
    if (!rcNumber || !/^\d+$/.test(rcNumber)) {
      console.log("RC number is empty or invalid, setting to 1");
      rcNumber = 1;
    } else {
      console.log(`Current RC number: ${rcNumber}`);
      rcNumber = parseInt(rcNumber) + 1;
    }
    newVersion = `${baseVersion}-rc.${rcNumber}`;
    console.log(`New version for minor bump when minor was true: ${newVersion}`);
  }

  return { newBumpInfo, newVersion };
}

function handlePatchBump(currentBumpInfo, major, minor, patch, baseVersion, rcNumber) {
  let newBumpInfo = JSON.stringify({ ...currentBumpInfo, patch: true });
  console.log(`New bump info for patch: ${newBumpInfo}`);
  let newVersion;

  if (!currentBumpInfo.patch) {
    newVersion = `${major}.${minor}.${parseInt(patch) + 1}-rc.1`;
    console.log(`New version for patch bump when patch was false: ${newVersion}`);
  } else {
    if (!rcNumber || !/^\d+$/.test(rcNumber)) {
      console.log("RC number is empty or invalid, setting to 1");
      rcNumber = 1;
    } else {
      console.log(`Current RC number: ${rcNumber}`);
      rcNumber = parseInt(rcNumber) + 1;
    }
    newVersion = `${baseVersion}-rc.${rcNumber}`;
    console.log(`New version for patch bump when patch was true: ${newVersion}`);
  }

  return { newBumpInfo, newVersion };
}

async function run() {
  try {
    const latestRelease = extractLatestRelease(core.getInput('latest_release'));
    const version = JSON.parse(latestRelease).version;
    const bumpType = core.getInput('bump_type');
    const currentBumpInfo = JSON.parse(latestRelease).bumpInfo;

    console.group('Debug Information');
    console.log(`Current version: ${version}`);
    console.log(`Bump type: ${bumpType}`);
    console.log(`Current bump info: ${JSON.stringify(currentBumpInfo)}`);
    console.groupEnd();

    const versionRegex = /^([0-9]+)\.([0-9]+)\.([0-9]+)(-rc\.([0-9]+))?$/;
    const match = version.match(versionRegex);

    if (!match) {
      throw new Error("Version format is invalid.");
    }

    const [_, major, minor, patch, __, rcNumber] = match;
    const baseVersion = `${major}.${minor}.${patch}`;

    console.log(`Base version: ${baseVersion}`);
    console.log(`RC number: ${rcNumber}`);

    let result;
    switch (bumpType) {
      case 'major':
        result = handleMajorBump(currentBumpInfo, major);
        break;
      case 'minor':
        result = handleMinorBump(currentBumpInfo, major, minor, baseVersion, rcNumber);
        break;
      default:
        result = handlePatchBump(currentBumpInfo, major, minor, patch, baseVersion, rcNumber);
        break;
    }

    console.log(`Final new version: ${result.newVersion}`);
    console.log(`Final new bump info: ${result.newBumpInfo}`);
    core.setOutput('new_version', result.newVersion);
    core.setOutput('bump_info', result.newBumpInfo);

  } catch (error) {
    core.setFailed(error.message);
  }
}

run();
