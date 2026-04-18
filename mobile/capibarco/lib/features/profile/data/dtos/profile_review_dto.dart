import '../../domain/entities/profile_review.dart';

class ProfileReviewDto {
  const ProfileReviewDto({
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

  factory ProfileReviewDto.fromJson(Map<String, dynamic> json) {
    final audit = json['audit'] as Map<String, dynamic>? ?? const {};
    return ProfileReviewDto(
      id: json['review_id'] as String? ?? '',
      authorProfileId: json['author_profile_id'] as String? ?? '',
      rating: (json['rating'] as num?)?.toInt() ?? 0,
      text: json['text'] as String? ?? '',
      createdAt: DateTime.tryParse(audit['created_at'] as String? ?? ''),
    );
  }

  ProfileReviewEntity toDomain() {
    return ProfileReviewEntity(
      id: id,
      authorProfileId: authorProfileId,
      rating: rating,
      text: text,
      createdAt: createdAt,
    );
  }
}
