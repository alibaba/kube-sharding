import sys, os
import subprocess
# usage:
#   python2 register_crd.py path_to_kubeconfig path_to_crd_yaml_dir
kubeconfig = sys.argv[1]
crd_yaml_dir = sys.argv[2]

crd_yaml_files = [f for f in os.listdir(crd_yaml_dir) if os.path.isfile(os.path.join(crd_yaml_dir, f))]

for crd in crd_yaml_files:
    crd_path = os.path.join(crd_yaml_dir, crd)
    print crd_path
    subprocess.call(["kubectl", "--kubeconfig", kubeconfig, "apply", "-f", crd_path])