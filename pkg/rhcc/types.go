package rhcc

type ContainerRepository struct {
	Entity        string `json:"entity"`
	EntityVersion string `json:"entityVersion"`
	Status        string `json:"status"`
	ModifiedCount int    `json:"modifiedCount"`
	MatchCount    int    `json:"matchCount"`
	Hostname      string `json:"hostname"`
	Processed     []struct {
		Repository  string `json:"repository"`
		ProductID   string `json:"product_id"`
		DisplayData struct {
			ShortDescription string `json:"short_description"`
			Name             string `json:"name"`
			LongDescription  string `json:"long_description"`
			OpenshiftTags    string `json:"openshift_tags"`
		} `json:"display_data"`
		AutoRebuildTags         []string `json:"auto_rebuild_tags"`
		ProtectedForPull        bool     `json:"protected_for_pull"`
		ContentStreamTags       []string `json:"content_stream_tags"`
		Registry                string   `json:"registry"`
		ReleaseCategories       []string `json:"release_categories"`
		VendorLabel             string   `json:"vendorLabel"`
		PrivilegedImagesAllowed bool     `json:"privileged_images_allowed"`
		Vendors                 []struct {
			Label string `json:"label"`
			Name  string `json:"name"`
		} `json:"vendors"`
		Products []*Product `json:"products"`
		Images   []struct {
			LastUpdateDate  string            `json:"lastUpdateDate"`
			FreshnessGrades []*FreshnessGrade `json:"freshness_grades"`
			Repositories    []struct {
				ImageAdvisoryID string `json:"image_advisory_id"`
				Repository      string `json:"repository"`
				Tags            []struct {
					AddedDate  string `json:"added_date"`
					Name       string `json:"name"`
					TagHistory []struct {
						TagType string `json:"tag_type"`
					} `json:"tag_history"`
				} `json:"tags"`
				PushDate         string `json:"push_date"`
				ImageAdvisoryRef []struct {
					ShipDate string `json:"ship_date"`
				} `json:"image_advisory_ref"`
			} `json:"repositories"`
			TopLayerID    string `json:"top_layer_id"`
			DockerImageID string `json:"docker_image_id"`
			Architecture  string `json:"architecture"`
			CpeIds        int    `json:"cpe_ids#"`
			ID            string `json:"_id"`
		} `json:"images"`
	} `json:"processed"`
	ResultMetadata []interface{} `json:"resultMetadata"`
}

type FreshnessGrade struct {
	EndDate      string `json:"end_date,omitempty"`
	Grade        string `json:"grade"`
	CreationDate string `json:"creation_date"`
	StartDate    string `json:"start_date"`
}

type CVE struct {
	Severity string `json:"severity"`
	Packages []struct {
		SrpmNevra string   `json:"srpm_nevra"`
		RpmNvra   []string `json:"rpm_nvra"`
	} `json:"packages"`
	AdvisoryID string `json:"advisory_id"`
	CveID      string `json:"cve_id"`
}

type Product struct {
	ShortDescription        string   `json:"short_description"`
	SupportPolicyUrls       []string `json:"support_policy_urls"`
	Keywords                []string `json:"keywords"`
	ReleasePolicyUrls       []string `json:"release_policy_urls"`
	LastUpdateDate          string   `json:"lastUpdateDate"`
	BuildCategories         int      `json:"build_categories#"`
	TeamID                  string   `json:"team_id"`
	ObjectType              string   `json:"objectType"`
	LongDescriptionMarkdown string   `json:"long_description_markdown"`
	DocumentationLinks      int      `json:"documentation_links#"`
	KeywordsHidden          int      `json:"keywords_hidden#"`
	ApplicationCategories   []string `json:"application_categories"`
	LongDescription         string   `json:"long_description"`
	Published               bool     `json:"published"`
	CreationDate            string   `json:"creationDate"`
	URL                     string   `json:"url"`
	Industries              int      `json:"industries#"`
	VendorLabel             string   `json:"vendorLabel"`
	Name                    string   `json:"name"`
	ID                      string   `json:"_id"`
}

type ContainerRepositoryImage struct {
	Entity        string `json:"entity"`
	EntityVersion string `json:"entityVersion"`
	Status        string `json:"status"`
	ModifiedCount int    `json:"modifiedCount"`
	MatchCount    int    `json:"matchCount"`
	Hostname      string `json:"hostname"`
	Processed     []struct {
		Repository  string `json:"repository"`
		ProductID   string `json:"product_id"`
		DisplayData struct {
			LongDescriptionMarkdown string `json:"long_description_markdown"`
			ShortDescription        string `json:"short_description"`
			Name                    string `json:"name"`
			LongDescription         string `json:"long_description"`
			OpenshiftTags           string `json:"openshift_tags"`
		} `json:"display_data"`
		AutoRebuildTags         []string `json:"auto_rebuild_tags"`
		ProtectedForPull        bool     `json:"protected_for_pull"`
		ContentStreamTags       []string `json:"content_stream_tags"`
		Registry                string   `json:"registry"`
		ReleaseCategories       []string `json:"release_categories"`
		VendorLabel             string   `json:"vendorLabel"`
		PrivilegedImagesAllowed bool     `json:"privileged_images_allowed"`
		Vendors                 []struct {
			Label string `json:"label"`
			Name  string `json:"name"`
		} `json:"vendors"`
		Products []*Product `json:"products"`
		Images   []struct {
			LastUpdateDate  string `json:"lastUpdateDate"`
			FreshnessGrades []struct {
				EndDate      string `json:"end_date,omitempty"`
				Grade        string `json:"grade"`
				CreationDate string `json:"creation_date"`
				StartDate    string `json:"start_date"`
			} `json:"freshness_grades"`
			Repositories []struct {
				Comparison struct {
					Reason string `json:"reason"`
					Rpms   struct {
						New       []interface{} `json:"new"`
						Upgrade   []string      `json:"upgrade"`
						Downgrade []interface{} `json:"downgrade"`
						Remove    []string      `json:"remove"`
					} `json:"rpms"`
					AdvisoryRpmMapping []struct {
						Nvra        string   `json:"nvra"`
						AdvisoryIds []string `json:"advisory_ids"`
					} `json:"advisory_rpm_mapping"`
					WithNvr    string `json:"with_nvr"`
					ReasonText string `json:"reason_text"`
				} `json:"comparison"`
				ImageAdvisoryID string `json:"image_advisory_id"`
				Repository      string `json:"repository"`
				Signatures      []struct {
					KeyLongID string   `json:"key_long_id"`
					Tags      []string `json:"tags"`
				} `json:"signatures"`
				Tags []struct {
					AddedDate string `json:"added_date"`
					Name      string `json:"name"`
				} `json:"tags"`
				PushDate string `json:"push_date"`
			} `json:"repositories"`
			TopLayerID        string `json:"top_layer_id"`
			SumLayerSizeBytes int    `json:"sum_layer_size_bytes"`
			DockerImageID     string `json:"docker_image_id"`
			Architecture      string `json:"architecture"`
			CpeIds            int    `json:"cpe_ids#"`
			ID                string `json:"_id"`
			ParsedData        struct {
				Size  int `json:"size"`
				Files []struct {
					ContentURL string `json:"content_url"`
				} `json:"files"`
			} `json:"parsed_data"`
			VulnerabilitiesRef []struct {
				Severity string `json:"severity"`
				Packages []struct {
					SrpmNevra string   `json:"srpm_nevra"`
					RpmNvra   []string `json:"rpm_nvra"`
				} `json:"packages"`
				AdvisoryID string `json:"advisory_id"`
				CveID      string `json:"cve_id"`
			} `json:"vulnerabilitiesRef"`
			RpmManifest []struct {
				LastUpdatedBy string `json:"lastUpdatedBy"`
				Rpms          []struct {
					Summary      string `json:"summary"`
					SrpmName     string `json:"srpm_name"`
					Nvra         string `json:"nvra"`
					Release      string `json:"release"`
					SrpmNevra    string `json:"srpm_nevra"`
					Name         string `json:"name"`
					Version      string `json:"version"`
					Architecture string `json:"architecture"`
					Gpg          string `json:"gpg"`
					RedhatSigned bool   `json:"redhat_signed"`
				} `json:"rpms"`
				CreatedBy      string `json:"createdBy"`
				LastUpdateDate string `json:"lastUpdateDate"`
				ID             string `json:"_id"`
				CreationDate   string `json:"creationDate"`
				ImageID        string `json:"image_id"`
				ObjectType     string `json:"objectType"`
			} `json:"rpm_manifest"`
		} `json:"images"`
		LatestImage struct {
			Repositories []struct {
				Registry   string `json:"registry"`
				Repository string `json:"repository"`
				Tags       []struct {
					AddedDate string `json:"added_date"`
					Name      string `json:"name"`
				} `json:"tags"`
				PushDate string `json:"push_date"`
			} `json:"repositories"`
			TopLayerID    string `json:"top_layer_id"`
			DockerImageID string `json:"docker_image_id"`
			Architecture  string `json:"architecture"`
			ID            string `json:"_id"`
		} `json:"latest_image"`
	} `json:"processed"`
	ResultMetadata []interface{} `json:"resultMetadata"`
}
