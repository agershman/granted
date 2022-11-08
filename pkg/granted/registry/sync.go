package registry

import (
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

// TODO: Sync command/Add command should create new aws config file if not found.
var SyncCommand = cli.Command{
	Name:        "sync",
	Description: "Pull the latest change from remote origin and sync aws profiles in aws config files. For more click here https://github.com/common-fate/rfds/discussions/2",
	Action: func(c *cli.Context) error {

		gConf, err := grantedConfig.Load()
		if err != nil {
			return err
		}

		if len(gConf.ProfileRegistryURLS) < 1 {
			clio.Warn("granted registry not configured. Try adding a git repository with 'granted registry add <https://github.com/your-org/your-registry.git>'")
		}

		awsConfigPath, err := getDefaultAWSConfigLocation()
		if err != nil {
			return err
		}

		configFile, err := loadAWSConfigFile()
		if err != nil {
			return err
		}

		// THINKING: Maybe temporarily write stuff to tmp file
		// if unhandle error occurs after some content overwrite is done then revert to this file.
		// tmp, err := os.CreateTemp("/Users/eddie/dev/commonfate/granted", "config_backup")
		// if err != nil {
		// 	return err
		// }
		// configFile.WriteTo(tmp)
		// defer os.Remove(tmp.Name())

		// if the config file contains granted generated content then remove it
		if err := removeAutogeneratedProfiles(configFile, awsConfigPath); err != nil {
			return err
		}

		for index, repoURL := range gConf.ProfileRegistryURLS {
			u, err := parseGitURL(repoURL)
			if err != nil {
				return err
			}

			repoDirPath, err := getRegistryLocation(u)
			if err != nil {
				return err
			}

			if err = gitPull(repoDirPath, false); err != nil {
				return err
			}

			if err = parseClonedRepo(repoDirPath, repoURL); err != nil {
				return err
			}

			var r Registry
			_, err = r.Parse(repoDirPath)
			if err != nil {
				return err
			}

			isFirstSection := false
			if index == 0 {
				isFirstSection = true
			}

			// TODO: If it fails to sync for one repo; then should skip
			// but should print error at last.
			if err := Sync(r, repoURL, repoDirPath, isFirstSection); err != nil {
				return err
			}
		}

		return nil
	},
}

func Sync(r Registry, repoURL string, repoDirPath string, isFirstSection bool) error {
	clio.Debugf("syncing %s \n", repoURL)

	awsConfigPath, err := getDefaultAWSConfigLocation()
	if err != nil {
		return err
	}

	awsConfigFile, err := loadAWSConfigFile()
	if err != nil {
		return err
	}

	clonedFile, err := loadClonedConfigs(r, repoDirPath)
	if err != nil {
		return err
	}

	err = generateNewRegistrySection(awsConfigFile, clonedFile, repoURL, isFirstSection)
	if err != nil {
		return err
	}

	err = awsConfigFile.SaveTo(awsConfigPath)
	if err != nil {
		return err
	}

	clio.Debugln("Changes saved to aws config file.")

	return nil
}
