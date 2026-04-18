class ProfileReviewEntity {
  const ProfileReviewEntity({
    required this.id,
    required this.authorProfileId,
    required this.rating,
    required this.text,
    required this.createdAt,
  });

  final String id;
  final String authorProfileId;
  final int rating;
  final String text;
  final DateTime? createdAt;
}
