package runner

import (
	"crypto/tls"
	"fmt"
	"slices"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/z46-dev/freeipa-runner/config"
)

func getUsernameDN(username string) string {
	return fmt.Sprintf(
		"uid=%s,cn=%s,cn=%s,dc=%s,dc=%s",
		username,
		config.Config.LDAP.UsersCN,
		config.Config.LDAP.AccountsCN,
		config.Config.LDAP.DomainSLD,
		config.Config.LDAP.DomainTLD,
	)
}

func getUserGroupDN(groupName string) string {
	return fmt.Sprintf(
		"cn=%s,cn=%s,cn=%s,dc=%s,dc=%s",
		groupName,
		config.Config.LDAP.GroupsCN,
		config.Config.LDAP.AccountsCN,
		config.Config.LDAP.DomainSLD,
		config.Config.LDAP.DomainTLD,
	)
}

func getHostDN(fqdn string) string {
	return fmt.Sprintf(
		"fqdn=%s,cn=%s,cn=%s,dc=%s,dc=%s",
		fqdn,
		config.Config.LDAP.HostsCN,
		config.Config.LDAP.AccountsCN,
		config.Config.LDAP.DomainSLD,
		config.Config.LDAP.DomainTLD,
	)
}

func getHostGroupDN(groupName string) string {
	return fmt.Sprintf(
		"cn=%s,cn=%s,cn=%s,dc=%s,dc=%s",
		groupName,
		config.Config.LDAP.HostGroupsCN,
		config.Config.LDAP.AccountsCN,
		config.Config.LDAP.DomainSLD,
		config.Config.LDAP.DomainTLD,
	)
}

func getServiceDN(krbPrincipal string) string {
	return fmt.Sprintf(
		"krbprincipalname=%s,cn=%s,cn=%s,dc=%s,dc=%s",
		krbPrincipal,
		config.Config.LDAP.ServicesCN,
		config.Config.LDAP.AccountsCN,
		config.Config.LDAP.DomainSLD,
		config.Config.LDAP.DomainTLD,
	)
}

func getHostGroupsBaseDN() string {
	return fmt.Sprintf(
		"cn=%s,cn=%s,dc=%s,dc=%s",
		config.Config.LDAP.HostGroupsCN,
		config.Config.LDAP.AccountsCN,
		config.Config.LDAP.DomainSLD,
		config.Config.LDAP.DomainTLD,
	)
}

func Dial(username, password string) (socket *ldap.Conn, err error) {
	if socket, err = ldap.DialURL(config.Config.LDAP.Address, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true})); err != nil {
		return
	}

	if err = socket.Bind(getUsernameDN(username), password); err != nil {
		return
	}

	return
}

func GetGroupHosts(groupName string) (fqdns []string, err error) {
	var sock *ldap.Conn

	if sock, err = Dial(config.Config.LDAP.BindUsername, config.Config.LDAP.BindPassword); err != nil {
		return
	}

	defer sock.Close()

	var (
		base, filter string              = getHostGroupsBaseDN(), fmt.Sprintf("(cn=%s)", ldap.EscapeFilter(groupName))
		req          *ldap.SearchRequest = ldap.NewSearchRequest(
			base,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0, 0, false,
			filter,
			[]string{"memberHost", "memberHostGroup", "member"},
			nil,
		)
	)

	var searchResult *ldap.SearchResult
	if searchResult, err = sock.Search(req); err != nil {
		return
	}

	if len(searchResult.Entries) == 0 {
		err = fmt.Errorf("host group not found: %s", groupName)
		return
	}

	var (
		entry                   *ldap.Entry = searchResult.Entries[0]
		hostDNs, nestedGroupDNs []string    = entry.GetAttributeValues("memberHost"), entry.GetAttributeValues("memberHostGroup")
	)

	for _, dn := range entry.GetAttributeValues("member") {
		if strings.HasPrefix(strings.ToLower(dn), "fqdn=") {
			hostDNs = append(hostDNs, dn)
			continue
		}

		if strings.Contains(strings.ToLower(dn), fmt.Sprintf("cn=%s,", config.Config.LDAP.HostGroupsCN)) {
			nestedGroupDNs = append(nestedGroupDNs, dn)
		}
	}

	for _, groupDn := range nestedGroupDNs {
		var groupSearchResult *ldap.SearchResult
		if groupSearchResult, err = sock.Search(ldap.NewSearchRequest(
			groupDn, ldap.ScopeBaseObject, ldap.NeverDerefAliases,
			0, 0, false, "(objectClass=ipaHostGroup)",
			[]string{"memberHost", "member", "memberHostGroup"}, nil,
		)); err == nil && len(groupSearchResult.Entries) == 1 {
			hostDNs = append(hostDNs, groupSearchResult.Entries[0].GetAttributeValues("memberHost")...)
			for _, dn := range groupSearchResult.Entries[0].GetAttributeValues("member") {
				if strings.HasPrefix(strings.ToLower(dn), "fqdn=") {
					hostDNs = append(hostDNs, dn)
				}
			}
		}
	}

	for _, dn := range hostDNs {
		var fqdn string
		if fqdn = dnAttr(dn, "fqdn"); fqdn != "" {
			if !slices.Contains(fqdns, fqdn) {
				fqdns = append(fqdns, fqdn)
			}
		} else {
			var hostSearchRequest *ldap.SearchResult
			if hostSearchRequest, err = sock.Search(ldap.NewSearchRequest(
				dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases,
				0, 0, false, "(objectClass=ipaHost)",
				[]string{"fqdn"}, nil,
			)); err == nil && len(hostSearchRequest.Entries) == 1 {
				fq := hostSearchRequest.Entries[0].GetAttributeValue("fqdn")
				if fq != "" {
					if !slices.Contains(fqdns, fq) {
						fqdns = append(fqdns, fq)
					}
				}
			}
		}
	}

	return
}

func dnAttr(dn, key string) (value string) {
	key = strings.ToLower(key) + "="
	var parts []string = strings.Split(dn, ",")

	if len(parts) == 0 {
		return
	}

	var rdn string = strings.ToLower(strings.TrimSpace(parts[0]))
	if strings.HasPrefix(rdn, key) {
		value = strings.TrimSpace(parts[0][len(key):])
	}

	return
}
