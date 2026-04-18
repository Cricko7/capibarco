class ProfileSummaryEntity {
  const ProfileSummaryEntity({
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

class ProfileSummaryPage {
  const ProfileSummaryPage({
    required this.items,
    required this.nextPageToken,
    required this.isStale,
  });

  final List<ProfileSummaryEntity> items;
  final String nextPageToken;
  final bool isStale;
}
