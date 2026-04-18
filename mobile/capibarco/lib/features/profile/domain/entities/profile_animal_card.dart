class ProfileAnimalCardEntity {
  const ProfileAnimalCardEntity({
    required this.id,
    required this.name,
    required this.statusCode,
    required this.speciesLabel,
    required this.breed,
    required this.description,
    required this.city,
    required this.photoUrl,
    required this.statusLabel,
  });

  final String id;
  final String name;
  final String statusCode;
  final String speciesLabel;
  final String breed;
  final String description;
  final String city;
  final String photoUrl;
  final String statusLabel;

  bool get isDraft => statusCode == 'ANIMAL_STATUS_DRAFT';
  bool get isPublished => statusCode == 'ANIMAL_STATUS_AVAILABLE';
  bool get hasPhoto => photoUrl.isNotEmpty;
}
