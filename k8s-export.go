package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/mvcc/mvccpb"

	admissionv1b1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalev1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1b1 "k8s.io/api/batch/v1beta1"
	coordv1b1 "k8s.io/api/coordination/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extv1b1 "k8s.io/api/extensions/v1beta1"
	netv1 "k8s.io/api/networking/v1"
	netv1b1 "k8s.io/api/networking/v1beta1"
	policyv1b1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1b1 "k8s.io/api/storage/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

var dbPath string
var outPath string

type Unmarshallable interface {
	Unmarshal(bytes []byte) error
}

type revision struct {
	main int64
	sub  int64
}

func bytesToRev(bytes []byte) revision {
	return revision{
		main: int64(binary.BigEndian.Uint64(bytes[0:8])),
		sub:  int64(binary.BigEndian.Uint64(bytes[9:])),
	}
}

func init() {
	flag.StringVar(&dbPath, "db", "", "the path to the etcd database")
	flag.StringVar(&outPath, "output", "", "the path to the output folder")
	flag.StringVar(&outPath, "o", "", "the path to the output folder")
}

func isK8s0(b []byte) bool {
	return b[0] == 0x6b && b[1] == 0x38 && b[2] == 0x73 && b[3] == 0x00
}

func toYAML(obj interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return yaml.JSONToYAML(jsonBytes)
}

func vkYAML(obj *runtime.Unknown) ([]byte, error) {
	jsonBytes, err := json.Marshal(obj.TypeMeta)
	if err != nil {
		return nil, err
	}

	return yaml.JSONToYAML(jsonBytes)
}

func yamlFromBytes(bytes []byte, obj Unmarshallable) ([]byte, error) {
	err := obj.Unmarshal(bytes)
	if err != nil {
		return nil, err
	}
	return toYAML(obj)
}

func writeYAML(yamlFile string, obj *runtime.Unknown) (err error) {
	var yamlData []byte

	if obj.APIVersion == "v1" {
		if obj.Kind == "ConfigMap" {
			var cfgMap corev1.ConfigMap
			yamlData, err = yamlFromBytes(obj.Raw, &cfgMap)
		} else if obj.Kind == "Namespace" {
			var ns corev1.Namespace
			yamlData, err = yamlFromBytes(obj.Raw, &ns)
		} else if obj.Kind == "Secret" {
			var sec corev1.Secret
			yamlData, err = yamlFromBytes(obj.Raw, &sec)
		} else if obj.Kind == "Service" {
			var srv corev1.Service
			yamlData, err = yamlFromBytes(obj.Raw, &srv)
		} else if obj.Kind == "ServiceAccount" {
			var sa corev1.ServiceAccount
			yamlData, err = yamlFromBytes(obj.Raw, &sa)
		} else if obj.Kind == "PersistentVolume" {
			var pv corev1.PersistentVolume
			yamlData, err = yamlFromBytes(obj.Raw, &pv)
		} else if obj.Kind == "PersistentVolumeClaim" {
			var pvc corev1.PersistentVolumeClaim
			yamlData, err = yamlFromBytes(obj.Raw, &pvc)
		} else if obj.Kind == "Endpoints" {
			var ep corev1.Endpoints
			yamlData, err = yamlFromBytes(obj.Raw, &ep)
		} else if obj.Kind == "Event" {
			var ev corev1.Event
			yamlData, err = yamlFromBytes(obj.Raw, &ev)
		} else if obj.Kind == "LimitRange" {
			var lr corev1.LimitRange
			yamlData, err = yamlFromBytes(obj.Raw, &lr)
		} else if obj.Kind == "Node" {
			var node corev1.Node
			yamlData, err = yamlFromBytes(obj.Raw, &node)
		} else if obj.Kind == "Pod" {
			var pod corev1.Pod
			yamlData, err = yamlFromBytes(obj.Raw, &pod)
		} else if obj.Kind == "RangeAllocation" {
			var ra corev1.RangeAllocation
			yamlData, err = yamlFromBytes(obj.Raw, &ra)
		} else if obj.Kind == "ResourceQuota" {
			var rq corev1.ResourceQuota
			yamlData, err = yamlFromBytes(obj.Raw, &rq)
		}
	} else if obj.APIVersion == "extensions/v1beta1" {
		if obj.Kind == "Ingress" {
			var ing extv1b1.Ingress
			yamlData, err = yamlFromBytes(obj.Raw, &ing)
		}
	} else if obj.APIVersion == "networking.k8s.io/v1" {
		if obj.Kind == "NetworkPolicy" {
			var np netv1.NetworkPolicy
			yamlData, err = yamlFromBytes(obj.Raw, &np)
		}
	} else if obj.APIVersion == "networking.k8s.io/v1beta1" {
		if obj.Kind == "Ingress" {
			var ing netv1b1.Ingress
			yamlData, err = yamlFromBytes(obj.Raw, &ing)
		}
	} else if obj.APIVersion == "batch/v1" {
		if obj.Kind == "Job" {
			var job batchv1.Job
			yamlData, err = yamlFromBytes(obj.Raw, &job)
		}
	} else if obj.APIVersion == "batch/v1beta1" {
		if obj.Kind == "CronJob" {
			var cjob batchv1b1.CronJob
			yamlData, err = yamlFromBytes(obj.Raw, &cjob)
		}
	} else if obj.APIVersion == "apps/v1" {
		if obj.Kind == "Deployment" {
			var depl appsv1.Deployment
			yamlData, err = yamlFromBytes(obj.Raw, &depl)
		} else if obj.Kind == "DaemonSet" {
			var ds appsv1.DaemonSet
			yamlData, err = yamlFromBytes(obj.Raw, &ds)
		} else if obj.Kind == "StatefulSet" {
			var ss appsv1.StatefulSet
			yamlData, err = yamlFromBytes(obj.Raw, &ss)
		} else if obj.Kind == "ControllerRevision" {
			var cr appsv1.ControllerRevision
			yamlData, err = yamlFromBytes(obj.Raw, &cr)
		} else if obj.Kind == "ReplicaSet" {
			var rs appsv1.ReplicaSet
			yamlData, err = yamlFromBytes(obj.Raw, &rs)
		}
	} else if obj.APIVersion == "rbac.authorization.k8s.io/v1" {
		if obj.Kind == "ClusterRole" {
			var cr rbacv1.ClusterRole
			yamlData, err = yamlFromBytes(obj.Raw, &cr)
		} else if obj.Kind == "ClusterRoleBinding" {
			var crb rbacv1.ClusterRoleBinding
			yamlData, err = yamlFromBytes(obj.Raw, &crb)
		} else if obj.Kind == "Role" {
			var role rbacv1.Role
			yamlData, err = yamlFromBytes(obj.Raw, &role)
		} else if obj.Kind == "RoleBinding" {
			var rb rbacv1.RoleBinding
			yamlData, err = yamlFromBytes(obj.Raw, &rb)
		}
	} else if obj.APIVersion == "admissionregistration.k8s.io/v1beta1" {
		if obj.Kind == "MutatingWebhookConfiguration" {
			var mwc admissionv1b1.MutatingWebhookConfiguration
			yamlData, err = yamlFromBytes(obj.Raw, &mwc)
		} else if obj.Kind == "ValidatingWebhookConfiguration" {
			var vwc admissionv1b1.ValidatingWebhookConfiguration
			yamlData, err = yamlFromBytes(obj.Raw, &vwc)
		}
	} else if obj.APIVersion == "autoscaling/v1" {
		if obj.Kind == "HorizontalPodAutoscaler" {
			var hpa autoscalev1.HorizontalPodAutoscaler
			yamlData, err = yamlFromBytes(obj.Raw, &hpa)
		}
	} else if obj.APIVersion == "coordination.k8s.io/v1beta1" {
		if obj.Kind == "Lease" {
			var lease coordv1b1.Lease
			yamlData, err = yamlFromBytes(obj.Raw, &lease)
		}
	} else if obj.APIVersion == "policy/v1beta1" {
		if obj.Kind == "PodDisruptionBudget" {
			var pdb policyv1b1.PodDisruptionBudget
			yamlData, err = yamlFromBytes(obj.Raw, &pdb)
		} else if obj.Kind == "PodSecurityPolicy" {
			var psp policyv1b1.PodSecurityPolicy
			yamlData, err = yamlFromBytes(obj.Raw, &psp)
		}
	} else if obj.APIVersion == "scheduling.k8s.io/v1" {
		if obj.Kind == "PriorityClass" {
			var pc schedv1.PriorityClass
			yamlData, err = yamlFromBytes(obj.Raw, &pc)
		}
	} else if obj.APIVersion == "storage.k8s.io/v1" {
		if obj.Kind == "StorageClass" {
			var sc storagev1.StorageClass
			yamlData, err = yamlFromBytes(obj.Raw, &sc)
		}
	} else if obj.APIVersion == "storage.k8s.io/v1beta1" {
		if obj.Kind == "CSINode" {
			var csin storagev1b1.CSINode
			yamlData, err = yamlFromBytes(obj.Raw, &csin)
		}
	}

	if err != nil {
		return err
	}

	if yamlData == nil {
		fmt.Printf("Unknown %s/%s\n", obj.APIVersion, obj.Kind)
		return nil
	}

	versionKindData, err := vkYAML(obj)
	if err != nil {
		return err
	}

	yamlDir := filepath.Dir(yamlFile)
	if err := os.MkdirAll(yamlDir, 0777); err != nil && !os.IsExist(err) {
		return err
	}
	file, err := os.Create(yamlFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte("---\n"))
	if err != nil {
		return err
	}
	_, err = file.Write(versionKindData)
	if err != nil {
		return err
	}
	_, err = file.Write(yamlData)
	return err
}

func main() {
	flag.Parse()

	if dbPath == "" || outPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Open the database file
	opts := bolt.Options{
		ReadOnly: true,
	}
	db, err := bolt.Open(dbPath, 0666, &opts)

	if err != nil {
		fmt.Fprintf(os.Stderr, "k8s-export: %v\n", err)
		os.Exit(1)
	}

	defer db.Close()

	// Export the data
	if err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("key"))

		if err := bucket.ForEach(func(k, v []byte) error {
			// Decode the entry
			rev := bytesToRev(k)

			var kv mvccpb.KeyValue
			if err := kv.Unmarshal(v); err != nil {
				return err
			}

			key := string(kv.Key)

			// Ignore things that aren't in the registry
			if !strings.HasPrefix(key, "/registry/") {
				return nil
			}

			var obj runtime.Unknown
			if len(kv.Value) < 4 || !isK8s0(kv.Value[:4]) {
				return nil
			}
			if err := obj.Unmarshal(kv.Value[4:]); err != nil {
				return err
			}

			suffix := fmt.Sprintf("-%d-%d.yaml", rev.main, rev.sub)
			yamlPath := filepath.Join(outPath,
				filepath.FromSlash(key)) + suffix

			writeYAML(yamlPath, &obj)

			return nil
		}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "k8s-export: %v\n", err)
		os.Exit(1)
	}
}
