package enforcement

import (
	"log"
	"os/exec"
)

func ShowWarning() {
	log.Println("Displaying Warning Message")

	psScript := `
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$form = New-Object System.Windows.Forms.Form
$form.Text = "COMPLIANCE WARNING"
$form.TopMost = $true
$form.WindowState = "Maximized"
$form.FormBorderStyle = "None"
$form.BackColor = "Red"
$form.Opacity = 0.8
$form.ShowInTaskbar = $false

$label = New-Object System.Windows.Forms.Label
$label.Text = "COMPLIANCE WARNING` + "`n" + `YOUR DEVICE IS OUT OF COMPLIANCE.` + "`n" + `PLEASE CONTACT IT SUPPORT IMMEDIATELY."
$label.Font = New-Object System.Drawing.Font("Arial", 48, [System.Drawing.FontStyle]::Bold)
$label.ForeColor = "White"
$label.AutoSize = $true
$label.TextAlign = "MiddleCenter"

# Center the label
$form.Add_Load({
    $label.Location = New-Object System.Drawing.Point(
        ($form.Width - $label.Width) / 2,
        ($form.Height - $label.Height) / 2
    )
})

$form.Controls.Add($label)

# Close on click for demo purposes (in prod, maybe make it harder to close)
$form.Add_Click({ $form.Close() })
$label.Add_Click({ $form.Close() })

[System.Windows.Forms.Application]::Run($form)
`

	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", psScript)
	err := cmd.Start() // Non-blocking
	if err != nil {
		log.Println("Failed to show warning:", err)
	}
}
