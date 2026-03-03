package enforcement

func (b *Bitlocker) enableNoTPMPolicy() {
	b.exec.Run(
		"reg",
		"add",
		`HKLM\SOFTWARE\Policies\Microsoft\FVE`,
		"/v", "EnableBDEWithNoTPM",
		"/t", "REG_DWORD",
		"/d", "1",
		"/f",
	)
}
