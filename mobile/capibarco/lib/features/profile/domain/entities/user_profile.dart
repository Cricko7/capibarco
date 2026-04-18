class UserProfileEntity {
  const UserProfileEntity({
    required this.id,
    required this.displayName,
    required this.bio,
    required this.city,
    required this.avatarUrl,
    required this.typeLabel,
    required this.averageRating,
    required this.reviewsCount,
  });

  final String id;
  final String displayName;
  final String bio;
  final String city;
  final String avatarUrl;
  final String typeLabel;
  final double averageRating;
  final int reviewsCount;
}
