package controller

// // Helper function to compare existing and desired Ingress objects
// func ingressEqual(existing, desired *networkingv1.Ingress) bool {
// 	// Compare labels and specs (you can add other fields if necessary)
// 	return reflect.DeepEqual(existing.Labels, desired.Labels) &&
// 		reflect.DeepEqual(existing.Spec, desired.Spec)
// }

// Utility functions to handle default values
func getOrDefaultString(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getOrDefaultInt(value, defaultValue int32) *int32 {
	if value == 0 {
		return &defaultValue
	}
	return &value
}
