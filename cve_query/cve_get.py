#Steps to take: 1. OCM log in 2. Get all cluster names/IDs for Registry Query to find current image hash 3. Get image IDs needed fo Pyxis query 3. Build and execute Queries 4. Save output to file 5. If any CVEs found print to terminal
import os, sys, getopt, subprocess
from pathlib import Path
# start in the directory this script is run from
#Path to certs
folder = Path('certs')
#extension = '.json'
#How we're saving the output
reports = Path("Report.json")
#filename = Path("output.txt")
reports.touch(exist_ok=True)
#Config files with clusters to run against would look something like this
# def files(path):
#     for file in os.listdir(path):
#         if os.path.isfile(os.path.join(path, file)):
#         	if file.endswith(extension):
# 	            yield os.path.join(path, file)
#Build query, execute curl, write output
def oc_auth():
    #This relies on having ocm-login-production as an alias for ocm login set up in your bash.rc file not needed to access pyxis
    login_cmd = ["ocm-login-production"]
    print("ocm session started")
def main():
    with open(reports, 'w') as logfile:
      #Require VPN conenctions:
        curl_cmd = ['curl', '--key', 'certs/hlipsig-demo-cert.key', '--cert', 'certs/hlipsig-demo-cert.crt', 'https://pyxis.engineering.redhat.com/v1/images','>', 'images.json']
          #https://pyxis.api.redhat.com/v1/ui/#/Advisories/pyxis.advisories.get_advisories requires VPN but here's API defintion docs
        curl_output = subprocess.Popen(curl_cmd, stdout=subprocess.PIPE)

        for line in curl_output.stdout.readlines():
                    # decode workaround required because python 3.5+ (◔_◔)
                    # avoids "TypeError: a bytes-like object is required, not 'str'"
         line = line.decode()
         logfile.write(line)
                    # print(len(line))





if __name__ == "__main__":
    main()
    #main(sys.argv[1:]) Ignore this for now.
    print('done!')
