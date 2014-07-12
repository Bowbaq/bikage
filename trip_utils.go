package main

type UserTrips []UserTrip

func partition(trips UserTrips, pass func(*UserTrip) bool) (UserTrips, UserTrips) {
	var in = make(UserTrips, 0)
	var out = make(UserTrips, 0)

	for _, s := range trips {
		if pass(&s) {
			in = append(in, s)
		} else {
			out = append(out, s)
		}
	}

	return in, out
}

func group_by(trips UserTrips, same func(*UserTrip, *UserTrip) bool) []UserTrips {
	var result = make([]UserTrips, 0)

	if len(trips) == 0 {
		return result
	}

	head := trips[0]
	group := trips[:1]
	trips = trips[1:]

	for _, s := range trips {
		if same(&head, &s) {
			group = append(group, s)
		} else {
			result = append(result, group)
			head = s
			group = make(UserTrips, 0)
			group = append(group, s)
		}
	}

	result = append(result, group)

	return result
}
