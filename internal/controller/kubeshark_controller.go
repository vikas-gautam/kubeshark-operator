package controller

import (
	"context"
	"fmt"

	kubesharkv1beta1 "kubeshark-operator/api/v1beta1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// KubesharkReconciler reconciles a Kubeshark object
type KubesharkReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kubeshark.opstreelabs.in,resources=kubesharks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kubeshark.opstreelabs.in,resources=kubesharks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kubeshark.opstreelabs.in,resources=kubesharks/finalizers,verbs=update

// Reconcile is part of the main Kubernetes reconciliation loop
func (r *KubesharkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Define the finalizer string
	const finalizerName = "kubeshark.finalizers.opstreelabs.in"

	// Fetch the Kubeshark instance
	cr := &kubesharkv1beta1.Kubeshark{}

	err := r.Client.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if errors.IsNotFound(err) {
			// CR was deleted, handle cleanup if necessary
			logger.Info("Kubeshark resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Kubeshark resource")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	logger.Info("Kubeshark resource found", "name", cr.Name, "namespace", cr.Namespace)
	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(cr, finalizerName) {
		controllerutil.AddFinalizer(cr, finalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Define labels to be used by all resources managed by this CR
	labels := map[string]string{
		"app.kubernetes.io/name":       "kubeshark",
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "kubeshark-operator",
	}

	hubConfigMap, resp, err := r.createOrUpdateHubConfigMap(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create hub Configmap")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", hubConfigMap.Name)

	// Reconcile Deployment
	hubDeployment, resp, err := r.createOrUpdateHubDeployment(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create hub Deployment")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", hubDeployment.Name)

	// Reconcile Service
	hubService, resp, err := r.createOrUpdateHubService(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create hub Service")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", hubService.Name)

	// Reconcile ServiceAccount
	serviceAccount, resp, err := r.createOrUpdateServiceAccount(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create ServiceAccount")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", serviceAccount.Name)

	// Reconcile Worker ServiceAccount
	workerServiceAccount, resp, err := r.createOrUpdateWorkerServiceAccount(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create Worker ServiceAccount")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", workerServiceAccount.Name)

	// Reconcile clusterrole
	clusterRole, resp, err := r.createOrUpdateClusterRole(labels)
	if err != nil {
		logger.Error(err, "Failed to create cluster role")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", clusterRole.Name)

	// Reconcile clusterrolebinding
	clusterRoleBinding, resp, err := r.createOrUpdateClusterRoleBinding(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create cluster rolebinding")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", clusterRoleBinding.Name)

	// Reconcile worker daemonset
	workerDaemonset, resp, err := r.createOrUpdateWorkerDaemonSet(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create worker daemonset")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", workerDaemonset.Name)

	// Reconcile frontend Deployment
	frontendDeployment, resp, err := r.createOrUpdateFrontendDeployment(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create frontendDeployment")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", frontendDeployment.Name)

	// Reconcile frontend service
	frontendService, resp, err := r.createOrUpdateFrontendService(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create frontendService")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", frontendService.Name)

	// Reconcile frontend ingress
	frontendIngress, resp, err := r.createOrUpdateFrontendIngress(cr, labels)
	if err != nil {
		logger.Error(err, "Failed to create frontendIngress")
		return ctrl.Result{}, err
	}
	logger.Info(resp, "name", frontendIngress.Name)

	// Handle deletion
	if !cr.ObjectMeta.DeletionTimestamp.IsZero() {
		// Perform cleanup for various resources

		if err := r.cleanupResource(ctx, cr, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: getOrDefaultString(cr.Spec.HubServiceName, "kubeshark-hub-service")}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for Service")

		if err := r.cleanupResource(ctx, cr, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: getOrDefaultString(cr.Spec.HubDeploymentName, "kubeshark-hub-deployment")}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for deployment")

		if err := r.cleanupResource(ctx, cr, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: getOrDefaultString(cr.Spec.HubConfigMapName, "kubeshark-hub-config")}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for configmap")

		serviceAccounts := []string{"kubeshark-service-account", "kubeshark-worker"}

		for _, saName := range serviceAccounts {
			sa := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      saName,
					Namespace: cr.Namespace, // Make sure namespace is set if needed
				},
			}
			if err := r.cleanupResource(ctx, cr, sa); err != nil {
				return ctrl.Result{}, err
			}
			logger.Info(fmt.Sprintf("Cleanup done for ServiceAccount: %s", saName))
		}

		if err := r.cleanupResource(ctx, cr, &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "kubeshark-cluster-role"}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for ClusterRole")

		if err := r.cleanupResource(ctx, cr, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "kubeshark-cluster-role-binding"}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for ClusterRoleBinding")

		if err := r.cleanupResource(ctx, cr, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "kubeshark-worker-daemonset"}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for worker daemonset")

		if err := r.cleanupResource(ctx, cr, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "kubeshark-frontend-service"}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for Frontend Service")

		if err := r.cleanupResource(ctx, cr, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "kubeshark-frontend-deployment"}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for Frontend Deployment")

		if err := r.cleanupResource(ctx, cr, &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "kubeshark-frontend-ingress"}}); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Cleanup done for frontend Ingress")

		// Remove finalizer once cleanup is complete
		controllerutil.RemoveFinalizer(cr, finalizerName)
		if err := r.Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("Removed finalizer from cr")

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
	
// SetupWithManager sets up the controller with the Manager.
func (r *KubesharkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubesharkv1beta1.Kubeshark{}).
		Owns(&appsv1.Deployment{}).         // Hub deployment
		Owns(&appsv1.DaemonSet{}).          // Worker DaemonSet
		Owns(&corev1.Service{}).            // Services for Hub/Worker
		Owns(&networkingv1.Ingress{}).      // Ingress
		Owns(&corev1.ConfigMap{}).          // ConfigMaps
		Owns(&corev1.ServiceAccount{}).     // SA for Hub/Worker
		Owns(&rbacv1.ClusterRole{}).        // ClusterRole
		Owns(&rbacv1.ClusterRoleBinding{}). // ClusterRoleBinding
		Complete(r)
}
