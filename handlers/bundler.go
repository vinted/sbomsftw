package handlers

const Gemfile = "Gemfile"
const GemfileLock = "Gemfile.lock"
const SBOMCollectionPrefix = "cyclonedx-ruby -p "

type Bundler struct{}

func (b Bundler) MatchFile(filename string) bool {
	return filename == Gemfile || filename == GemfileLock
}

func (b Bundler) GenerateBOM(string) (string, error) {

	//		err := os.Chdir(root)
	//		//Log error here
	//		if err != nil {
	//			return false
	//		}
	//		_, err = exec.Command("bundler install").Output()
	// out, err := exec.Command(SBOMCollectionPrefix + bootstrappedRoot).Output()
	return "", nil
}
