package syncclusterrolebinding

import (
	"context"
	"testing"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

var (
	scheme = runtime.NewScheme()
)

func generateClustersetToClusters(ms map[string]sets.String) *helpers.ClusterSetMapper {
	clustersetToClusters := helpers.NewClusterSetMapper()
	for s, c := range ms {
		clustersetToClusters.UpdateClusterSetByObjects(s, c)
	}
	return clustersetToClusters
}

func TestSyncManagedClusterClusterroleBinding(t *testing.T) {
	ca0 := generateRequiredClusterRoleBinding("c0", nil, "cs0", "admin")
	cv0 := generateRequiredClusterRoleBinding("c0", nil, "cs1", "view")
	cv1 := generateRequiredClusterRoleBinding("c1", nil, "cs2", "view")
	objs := []runtime.Object{ca0, cv0, cv1}
	kubeClient := k8sfake.NewSimpleClientset(objs...)

	ctc1 := generateClustersetToClusters(nil)

	ms2 := map[string]sets.String{"cs1": sets.NewString("c1", "c2")}
	ctc2 := generateClustersetToClusters(ms2)
	gs := map[string]sets.String{"global": sets.NewString("c1", "c2")}
	gsm := generateClustersetToClusters(gs)
	tests := []struct {
		name                   string
		clustersetToClusters   *helpers.ClusterSetMapper
		globalsetToClusters    *helpers.ClusterSetMapper
		clusterSetCache        *cache.AuthCache
		clustersetToSubject    map[string][]rbacv1.Subject
		clusterrolebindingName string
		exist                  bool
	}{
		{
			name:                 "no cluster",
			clustersetToClusters: ctc1,
			globalsetToClusters:  gsm,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetClusterRoleBindingName("c1", "admin"),
			exist:                  false,
		},
		{
			name:                 "delete c0:",
			clustersetToClusters: ctc1,
			globalsetToClusters:  gsm,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetClusterRoleBindingName("c0", "admin"),
			exist:                  false,
		},
		{
			name:                 "need create:",
			clustersetToClusters: ctc2,
			globalsetToClusters:  gsm,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetClusterRoleBindingName("c1", "admin"),
			exist:                  true,
		},
	}

	for _, test := range tests {
		ctx := context.Background()

		r := NewReconciler(kubeClient, test.clusterSetCache, test.clusterSetCache, test.globalsetToClusters, test.clustersetToClusters)
		r.reconcile()
		r.syncManagedClusterClusterroleBinding(ctx, test.clustersetToClusters, test.clustersetToSubject, "admin")
		validateResult(t, test.name, &r, test.clusterrolebindingName, test.exist)
	}
}

func validateResult(t *testing.T, caseName string, r *Reconciler, clusterrolebindingName string, exist bool) {
	ctx := context.Background()
	clusterrolebinding, _ := r.kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, clusterrolebindingName, metav1.GetOptions{})
	if exist && clusterrolebinding == nil {
		t.Errorf("Case: %v, Failed to apply clusterrolebinding", caseName)
	}
}

func Test_getClusterNameInClusterrolebinding(t *testing.T) {
	type args struct {
		clusterrolebindingName string
		role                   string
	}
	tests := []struct {
		name                   string
		clusterrolebindingName string
		want                   string
	}{
		{
			name:                   "right name",
			clusterrolebindingName: "open-cluster-management:managedclusterset:admin:managedcluster:managedcluster1",
			want:                   "managedcluster1",
		},
		{
			name:                   "wrong name",
			clusterrolebindingName: "",
			want:                   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getClusterNameInClusterrolebinding(tt.clusterrolebindingName); got != tt.want {
				t.Errorf("getClusterNameInClusterrolebinding() = %v, want %v", got, tt.want)
			}
		})
	}
}
