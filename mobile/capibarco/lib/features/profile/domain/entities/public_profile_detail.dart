import 'profile_animal_card.dart';
import 'profile_review.dart';
import 'user_profile.dart';

class PublicProfileDetailEntity {
  const PublicProfileDetailEntity({
    required this.profile,
    required this.reviews,
    required this.animals,
  });

  final UserProfileEntity profile;
  final List<ProfileReviewEntity> reviews;
  final List<ProfileAnimalCardEntity> animals;
}
