package bundle

import "csghub-airgap/internal/k3s"

func (s *Spec) Deploy() error {
	if err := checkArch(); err != nil {
		return err
	}

	if !s.SkipK3s {
		if err := k3s.Install(s.K3s); err != nil {
			return err
		}
	}

	return nil
}
