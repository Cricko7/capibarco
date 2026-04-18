class AnimalEditorEntity {
  const AnimalEditorEntity({
    required this.id,
    required this.name,
    required this.species,
    required this.breed,
    required this.sex,
    required this.size,
    required this.ageMonths,
    required this.description,
    required this.traits,
    required this.vaccinated,
    required this.sterilized,
    required this.city,
    required this.photoUrl,
    required this.status,
  });

  final String id;
  final String name;
  final String species;
  final String breed;
  final String sex;
  final String size;
  final int ageMonths;
  final String description;
  final List<String> traits;
  final bool vaccinated;
  final bool sterilized;
  final String city;
  final String photoUrl;
  final String status;

  bool get hasPhoto => photoUrl.isNotEmpty;
  bool get isDraft => status == 'ANIMAL_STATUS_DRAFT';
}
