/*
* Large-Scale Discovery, a network scanning solution for information gathering in large IT/OT network environments.
*
* Copyright (c) Siemens AG, 2016-2021.
*
* This work is licensed under the terms of the MIT license. For a copy, see the LICENSE file in the top-level
* directory or visit <https://opensource.org/licenses/MIT>.
*
 */

package config

import (
	"encoding/json"
	"fmt"
	scanUtils "github.com/siemens/GoScans/utils"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"large-scale-discovery/_build"
	"large-scale-discovery/log"
	"large-scale-discovery/manager/database"
	"large-scale-discovery/utils"
	"sync"
	"time"
)

var managerConfig = &ManagerConfig{} // Global configuration
var managerConfigLock sync.Mutex     // Lock required to avoid simultaneous requesting/updating of config

// Init initializes the configuration module and loads a JSON configuration. If JSON is not existing, a default
// configuration will be generated.
func Init(configFile string) error {
	if errFile := scanUtils.IsValidFile(configFile); errFile != nil {
		defaultConf := defaultManagerConfigFactory()
		errSave := Save(&defaultConf, configFile)
		if errSave != nil {
			return fmt.Errorf("could not initialize configuration in '%s': %s", configFile, errSave)
		} else {
			return fmt.Errorf("no configuration, created default in '%s'", configFile)
		}
	} else {
		errLoad := Load(configFile)
		if errLoad != nil {
			return errLoad
		} else {
			return nil
		}
	}
}

// Load reads a configuration from a file, de-serializes it and sets it as the global configuration
func Load(path string) error {

	// Lock global config before initializing an update
	managerConfigLock.Lock()
	defer managerConfigLock.Unlock()

	// Prepare new config, don't work on the global values
	newConfig := &ManagerConfig{}

	// Read file content
	rawJson, errLoad := ioutil.ReadFile(path)
	if errLoad != nil {
		return errLoad
	}

	// Parse Json
	errUnmarshal := json.Unmarshal(rawJson, newConfig)
	if errUnmarshal != nil {
		return errUnmarshal
	}

	// Replace global configuration with new one
	managerConfig = newConfig

	// Return nil to indicate successful config update
	return nil
}

// Set sets a passed configuration as the global configuration
func Set(conf *ManagerConfig) {

	// Lock global config before initializing an update
	managerConfigLock.Lock()
	defer managerConfigLock.Unlock()

	// Replace global configuration with new one
	managerConfig = conf
}

// Save serializes a given configuration and writes it to a file
func Save(conf *ManagerConfig, path string) error {

	// Lock global config, because the given config pointer might be the global config
	managerConfigLock.Lock()
	defer managerConfigLock.Unlock()

	// Serialize to Json
	file, errMarshal := json.MarshalIndent(conf, "", "    ")
	if errMarshal != nil {
		return errMarshal
	}

	// Write Json to file
	errWrite := ioutil.WriteFile(path, file, 0644)
	if errWrite != nil {
		return errWrite
	}

	// Return nil to indicate successful storage
	return nil
}

// GetConfig returns a pointer to the current global configuration.
func GetConfig() *ManagerConfig {

	// The global configuration might get updated regularly to allow user updating settings without aborting scans
	managerConfigLock.Lock()
	defer managerConfigLock.Unlock()

	// Return current global configuration
	return managerConfig
}

func defaultManagerConfigFactory() ManagerConfig {

	// Prepare default logging settings and adapt for manager
	logging := log.DefaultLogSettingsFactory()
	logging.File.Path = "./logs/manager.log"
	logging.Smtp.Connector.Subject = "Manager Error Log"

	// Prepare default settings for development
	if _build.DevMode {
		logging.Console.Level = zapcore.DebugLevel
	}

	// Define default values
	defaultInvalidPorts := []int{515, 631, 9100, 9101, 9102, 9103}
	defaultSkipDays := []time.Weekday{0, 6}
	defaultDiscoveryTimeEarliest := "09:00"
	defaultDiscoveryTimeLatest := "15:00"
	defaultNmapArgs := "-PE -PP -Pn -sS -sU -O -p U:53,67,68,161,162,1900,T:0-65535 -sV -T4 --randomize-hosts --host-timeout 6h --max-retries 2 --traceroute --script=default"

	// Ease some default values in development mode
	if _build.DevMode {
		defaultDiscoveryTimeEarliest = "00:00"
		defaultDiscoveryTimeLatest = "23:59"
		defaultNmapArgs = "-PE -PP -Pn -sS -O --top-ports 10 -sV -T4 --randomize-hosts --host-timeout 6h --max-retries 2 --traceroute"
	}

	// Prepare default manager config
	scanDefaults := database.T_scan_settings{
		MaxInstancesDiscovery:        30,
		MaxInstancesBanner:           100,
		MaxInstancesNfs:              0,
		MaxInstancesSmb:              10,
		MaxInstancesSsh:              25,
		MaxInstancesSsl:              25,
		MaxInstancesWebcrawler:       20,
		MaxInstancesWebenum:          25,
		SensitivePorts:               utils.JoinInt(defaultInvalidPorts, ","),
		SensitivePortsSlice:          defaultInvalidPorts,
		NetworkTimeoutSeconds:        8,
		HttpUserAgent:                "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0",
		DiscoveryTimeEarliest:        defaultDiscoveryTimeEarliest,
		DiscoveryTimeLatest:          defaultDiscoveryTimeLatest,
		DiscoverySkipDays:            utils.JoinWeekdays(defaultSkipDays, ","),
		DiscoverySkipDaysSlice:       defaultSkipDays,
		DiscoveryNmapArgs:            defaultNmapArgs,
		NfsScanTimeoutMinutes:        60 * 24 * 4,
		NfsDepth:                     -1,
		NfsThreads:                   4,
		NfsExcludeShares:             "",
		NfsExcludeFolders:            "",
		NfsExcludeExtensions:         "",
		NfsExcludeFileSizeBelow:      -1,
		NfsExcludeLastModifiedBelow:  time.Date(2008, 01, 01, 00, 00, 00, 00, time.UTC),
		NfsAccessibleOnly:            true,
		SmbScanTimeoutMinutes:        60 * 24 * 4,
		SmbDepth:                     -1,
		SmbThreads:                   4,
		SmbExcludeShares:             "print$,W7DP$,LSDP,LSDP_mosaic$,LSDP_test$,LSDP.WW005,lsdp-backup,lsdp_drivers_ww300$,LSDPW7$,custom_root$,gplmshare,SCCMContentLib$,SCCMContentLibD$,WsusContent",
		SmbExcludeFolders:            "",
		SmbExcludeExtensions:         "",
		SmbExcludeFileSizeBelow:      -1,
		SmbExcludeLastModifiedBelow:  time.Date(2008, 01, 01, 00, 00, 00, 00, time.UTC),
		SmbAccessibleOnly:            true,
		SslScanTimeoutMinutes:        30,
		SshScanTimeoutMinutes:        30,
		WebcrawlerScanTimeoutMinutes: 60 * 4,
		WebcrawlerDepth:              3,
		WebcrawlerMaxThreads:         4,
		WebcrawlerFollowQueryStrings: true,
		WebcrawlerAlwaysStoreRoot:    true,
		WebcrawlerFollowTypes:        "text/html,text/plain,text/javascript,application/javascript,application/json,application/atom+xml,application/rss+xml,application/xhtml+xml,application/x-latex,application/xml,application/xml-dtd,application/x-sh,application/x-tex,application/x-texinfo,text/cache-manifest,text/calendar,text/css,text/csv,text/csv-schema,text/directory,text/dns,text/ecmascript,text/encaprtp,text/example,text/fwdred,text/grammar-ref-list,text/jcr-cnd,text/markdown,text/mizar,text/n3,text/parameters,text/provenance-notation,text/prs.fallenstein.rst,text/prs.lines.tag,text/raptorfec,text/RED,text/rfc822-headers,text/rtf,text/rtp-enc-aescm128,text/rtploopback,text/rtx,text/SGML,text/t140,text/tab-separated-values,text/troff,text/turtle,text/ulpfec,text/uri-list,text/vcard,text/vnd.abc,text/vnd.debian.copyright,text/vnd.DMClientScript,text/vnd.dvb.subtitle,text/vnd.esmertec.theme-descriptor,text/vnd.fly,text/vnd.fmi.flexstor,text/vnd.graphviz,text/vnd.in3d.3dml,text/vnd.in3d.spot,text/vnd.IPTC.NewsML,text/vnd.IPTC.NITF,text/vnd.latex-z,text/vnd.motorola.reflex,text/vnd.ms-mediapackage,text/vnd.net2phone.commcenter.command,text/vnd.radisys.msml-basic-layout,text/vnd.si.uricatalogue,text/vnd.sun.j2me.app-descriptor,text/vnd.trolltech.linguist,text/vnd.wap.si,text/vnd.wap.sl,text/vnd.wap.wmlscript,text/vnd.wap-wml,text/vnd-a,text/vnd-curl,text/xml,text/xml-external-parsed-entity",
		WebenumScanTimeoutMinutes:    60,
		WebenumProbeRobots:           true,
	}

	// Prepare default config values
	privilegeSecrets := make([]string, 0, 4) // capacity 4 for unit test, because this is what Golang makes out of it
	var privilegeSecret string
	var passwordExpiry time.Duration
	var tokenExpiry time.Duration
	if _build.DevMode {
		privilegeSecrets = append(privilegeSecrets, "dev_secret")
		passwordExpiry = time.Hour * 24 * 365
		tokenExpiry = time.Hour * 24 * 365
	} else {
		privilegeSecret, _ = utils.GenerateToken(utils.AlphaNumCaseSymbol, 64)
		privilegeSecrets = append(privilegeSecrets, privilegeSecret)
		passwordExpiry = time.Hour * 12    // Should fit closely enough for one working day =)
		tokenExpiry = time.Hour * 24 * 365 // Used as a maximum possible value
	}

	// Generate manager config with default values
	conf := ManagerConfig{
		ListenAddress:    "localhost:2222",
		PrivilegeSecrets: privilegeSecrets,
		Database: Database{
			Connections:         30,
			ConnectionsClient:   5,
			PasswordExpiryHours: passwordExpiry.Hours(),
			PasswordExpiry:      passwordExpiry,
			TokenExpiryDays:     tokenExpiry.Hours() / 24,
			TokenExpiry:         tokenExpiry,
		},
		Logging:      logging,
		ScanDefaults: scanDefaults,
	}

	// Return generated config
	return conf
}

//
// JSON structure of configuration
//

type Database struct {
	Connections         int           `json:"connections"`           // Connections used by the manager component itself
	ConnectionsClient   int           `json:"connections_client"`    // Connections allowed for any user client
	PasswordExpiryHours float64       `json:"password_expiry_hours"` // Exact expiry interval of user access token
	PasswordExpiry      time.Duration `json:"-"`                     //
	TokenExpiryDays     float64       `json:"token_expiry_days"`     // Maximum allowed expiry interval for none user bound access token
	TokenExpiry         time.Duration `json:"-"`
}

// UnmarshalJSON reads a JSON file, validates values and populates the configuration struct
func (d *Database) UnmarshalJSON(b []byte) error {

	// Prepare temporary auxiliary data structure to load raw Json data
	type aux Database
	var raw aux

	// Unmarshal serialized Json into temporary auxiliary structure
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	// Do input validation
	if raw.PasswordExpiryHours <= 0 {
		return fmt.Errorf("invalid password expiry duration")
	}

	// Do input validation
	if raw.TokenExpiryDays <= 0 {
		return fmt.Errorf("invalid maximum token expiry duration")
	}

	// Copy loaded Json values to actual config
	*d = Database(raw)

	// Set unserializable values
	d.PasswordExpiry = time.Duration(raw.PasswordExpiryHours) * time.Hour
	d.TokenExpiry = time.Duration(raw.TokenExpiryDays) * (time.Hour * 24)

	// Return nil as everything is valid
	return nil
}

type ManagerConfig struct {
	// The root configuration object tying all configuration segments together.
	ListenAddress    string                   `json:"listen_address"`
	PrivilegeSecrets []string                 `json:"privilege_secrets"` // Tokens granting the privilege to query full scope details, including scope secret and the scope's database credentials. E.g. the web backend does not require these details, in contrast to the broker.
	Database         Database                 `json:"database"`
	Logging          log.Settings             `json:"logging"`
	ScanDefaults     database.T_scan_settings `json:"scan_defaults"`
}
