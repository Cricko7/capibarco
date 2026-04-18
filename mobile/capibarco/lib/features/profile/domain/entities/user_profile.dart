class UserProfileEntity {
  const UserProfileEntity({
    required this.id,
    required this.displayName,
    required this.bio,
    required this.city,
    required this.avatarUrl,
    required this.typeCode,
    required this.typeLabel,
    required this.averageRating,
    required this.reviewsCount,
  });

  final String id;
  final String displayName;
  final String bio;
  final String city;
  final String avatarUrl;
  final String typeCode;
  final String typeLabel;
  final double averageRating;
  final int reviewsCount;
}
